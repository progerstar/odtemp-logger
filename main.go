package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sstallion/go-hid"
)

const (
	OD_VID     = 0x0483
	OD_IOT_PID = 0xA26A

	HID_DATA_REPORT_ID  = 1
	HID_EVENT_REPORT_ID = 2
	HID_FW_REPORT_ID    = 3
	HID_CMD_REPORT_ID   = 4
	HID_UUID_REPORT_ID  = 5
	HID_CMD_REPORT_SIZE = 7

	HID_CMD_RST_UAPP = 0xF0
	HID_CMD_RST_DFU  = 0xF1
	HID_CMD_RST_STM  = 0xFA
	VERSION          = "1.5.1"

	readTimeout      = time.Second
	missedReadLimit  = 9
	minDeviceDelayMs = 1
	minSamplePeriod  = time.Millisecond
)

type SensorSample struct {
	Temperature float64
	Humidity    float64
	HasHumidity bool
	At          time.Time
	Gen         uint64
}

type DeviceState struct {
	dev                  *hid.Device
	found                bool
	connectionGeneration uint64
	connectionAlive      int32
	mutex                sync.Mutex
}

func (ds *DeviceState) setDevice(dev *hid.Device) {
	ds.mutex.Lock()
	ds.dev = dev
	ds.found = true
	atomic.AddUint64(&ds.connectionGeneration, 1)
	atomic.StoreInt32(&ds.connectionAlive, 1)
	ds.mutex.Unlock()
}

func (ds *DeviceState) clearDevice() {
	ds.mutex.Lock()
	ds.found = false
	atomic.StoreInt32(&ds.connectionAlive, 0)
	if ds.dev != nil {
		ds.dev.Close()
		ds.dev = nil
	}
	ds.mutex.Unlock()
}

func (ds *DeviceState) getDevice() (*hid.Device, bool) {
	ds.mutex.Lock()
	defer ds.mutex.Unlock()
	return ds.dev, ds.found
}

func (ds *DeviceState) isAlive() bool {
	return atomic.LoadInt32(&ds.connectionAlive) == 1
}

func (ds *DeviceState) getGeneration() uint64 {
	return atomic.LoadUint64(&ds.connectionGeneration)
}

func setDeviceInterval(dev *hid.Device, newInterval uint32) error {
	featureBuf := make([]byte, 64)
	featureBuf[0] = HID_CMD_REPORT_ID

	_, err := dev.GetFeatureReport(featureBuf)
	if err != nil {
		return fmt.Errorf("ошибка при чтении feature report: %w", err)
	}

	binary.LittleEndian.PutUint32(featureBuf[1:5], newInterval)

	_, err = dev.SendFeatureReport(featureBuf)
	if err != nil {
		return fmt.Errorf("ошибка при записи feature report: %w", err)
	}
	return nil
}

func getDeviceInterval(dev *hid.Device) (uint32, error) {
	featureBuf := make([]byte, 64)
	featureBuf[0] = HID_CMD_REPORT_ID

	_, err := dev.GetFeatureReport(featureBuf)
	if err != nil {
		return 0, fmt.Errorf("ошибка при чтении feature report: %w", err)
	}

	interval := binary.LittleEndian.Uint32(featureBuf[1:5])
	return interval, nil
}

func validatePeriod(period float64) (time.Duration, error) {
	if math.IsNaN(period) || math.IsInf(period, 0) || period <= 0 {
		return 0, fmt.Errorf("period должен быть положительным числом")
	}

	const maxDurationSeconds = float64(1<<63-1) / float64(time.Second)
	if period > maxDurationSeconds {
		return 0, fmt.Errorf("period слишком большой")
	}

	duration := time.Duration(period * float64(time.Second))
	if duration < minSamplePeriod {
		return 0, fmt.Errorf("period должен быть не меньше %.3f секунды", minSamplePeriod.Seconds())
	}

	return duration, nil
}

func deviceIntervalMilliseconds(period float64) uint32 {
	intervalMs := uint32(period * 1000)
	if intervalMs == 0 {
		return minDeviceDelayMs
	}
	return intervalMs
}

func deviceSamplePeriod(logPeriod, cloudPeriod time.Duration, cloudEnabled bool) time.Duration {
	if cloudEnabled && cloudPeriod < logPeriod {
		return cloudPeriod
	}
	return logPeriod
}

func configureFastDeviceInterval(ds *DeviceState, period float64) {
	if period >= 2 {
		return
	}

	dev, _ := ds.getDevice()
	if dev == nil {
		return
	}

	intervalMs := deviceIntervalMilliseconds(period)
	if err := setDeviceInterval(dev, intervalMs); err != nil {
		log.Println(err)
	}
	if interval, err := getDeviceInterval(dev); err == nil {
		log.Printf("Полученный интервал: %d ms\n", interval)
	}
}

func logSensorSample(sample SensorSample) {
	if sample.HasHumidity {
		log.Printf("Температура: %.2f°C; Влажность: %.2f%%\n", sample.Temperature, sample.Humidity)
	} else {
		log.Printf("Температура: %.2f°C\n", sample.Temperature)
	}
}

func offerSample(ch chan SensorSample, sample SensorSample) {
	select {
	case ch <- sample:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- sample:
		default:
		}
	}
}

func startPeriodicSampleRecorder(quit <-chan struct{}, ds *DeviceState, period time.Duration, record func(SensorSample)) chan SensorSample {
	sampleChan := make(chan SensorSample, 1)

	go func() {
		now := time.Now()
		firstTick := now.Truncate(period).Add(period)
		timer := time.NewTimer(time.Until(firstTick))
		defer timer.Stop()

		bySlot := make(map[int64]SensorSample)

		for {
			select {
			case <-quit:
				return
			case s := <-sampleChan:
				if ds.isAlive() && s.Gen == ds.getGeneration() {
					slotEnd := s.At.Truncate(period).Add(period)
					bySlot[slotEnd.UnixNano()] = s
				}
			case tickTime := <-timer.C:
				prevEndUnix := tickTime.Truncate(period).UnixNano()
				if s, ok := bySlot[prevEndUnix]; ok {
					if ds.isAlive() && s.Gen == ds.getGeneration() {
						record(s)
					}
				}
				// Удаляем и пропущенные слоты: если record() блокировался
				// дольше периода, их тики уже не наступят
				for slotEnd := range bySlot {
					if slotEnd <= prevEndUnix {
						delete(bySlot, slotEnd)
					}
				}
				next := tickTime.Add(period)
				timer.Reset(time.Until(next))
			}
		}
	}()

	return sampleChan
}

func sendBootloaderCommand() error {
	dev, err := findAndOpenDevice()
	if err != nil {
		return err
	}

	// Формируем команду: Report ID + Command
	cmdBuf := make([]byte, HID_CMD_REPORT_SIZE)
	cmdBuf[0] = HID_CMD_REPORT_ID
	cmdBuf[1] = HID_CMD_RST_DFU

	_, err = dev.Write(cmdBuf)
	dev.Close()

	if err != nil {
		return fmt.Errorf("ошибка отправки команды: %w", err)
	}

	log.Println("Команда перехода в загрузчик отправлена")
	return nil
}

func findAndOpenDevice() (*hid.Device, error) {
	if err := hid.Init(); err != nil {
		return nil, fmt.Errorf("ошибка инициализации HID: %v", err)
	}

	var devices []*hid.DeviceInfo

	err := hid.Enumerate(OD_VID, OD_IOT_PID, func(info *hid.DeviceInfo) error {
		devices = append(devices, info)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("ошибка при перечислении устройств: %v", err)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("устройства не найдены")
	}

	deviceInfo := devices[0]
	devicePath := deviceInfo.Path

	log.Printf("Найдено устройство по пути: %s\n", devicePath)
	log.Printf("Открываем устройство...\n")

	dev, err := hid.OpenPath(devicePath)
	if err != nil {
		return nil, fmt.Errorf("невозможно открыть устройство: %v", err)
	}

	return dev, nil
}

func processDataReport(data []byte) (float64, float64, bool, bool) {
	if len(data) < 2 {
		return 0, 0, false, false
	}

	rawTemp := int16(binary.LittleEndian.Uint16(data[:2]))
	temp := float64(rawTemp) / 100.0

	if len(data) >= 4 {
		rawHumidity := int16(binary.LittleEndian.Uint16(data[2:4]))
		return temp, float64(rawHumidity) / 100.0, true, true
	}

	return temp, 0, false, true
}

func searchDevice(ds *DeviceState, quit <-chan struct{}, silent bool) bool {
	for {
		select {
		case <-quit:
			return false
		default:
		}

		dev, err := findAndOpenDevice()
		if err == nil {
			select {
			case <-quit:
				dev.Close()
				return false
			default:
			}

			ds.setDevice(dev)
			log.Println("Устройство успешно открыто")
			return true
		}

		if !silent {
			log.Println(err)
			log.Println("Повторная попытка через 1 сек...")
		}

		select {
		case <-quit:
			return false
		case <-time.After(time.Second):
		}
	}
}

func main() {
	cliPtr := flag.Bool("cli", false, "запуск без GUI")
	pathPtr := flag.String("path", "", "переопределяет путь записи лога")
	silentPtr := flag.Bool("silent", false, "не писать лог")
	periodPtr := flag.Float64("period", 60, "период записи в секундах")
	cloudTokenPtr := flag.String("cloud-token", "", "write_token для записи показаний в cloud-lite (или env ODTEMP_CLOUD_TOKEN)")
	cloudPeriodPtr := flag.Float64("cloud-period", 60, "период записи показаний в cloud-lite в секундах (или env ODTEMP_CLOUD_PERIOD)")
	cloudURLPtr := flag.String("cloud-url", defaultCloudBaseURL, "base URL cloud-lite (или env ODTEMP_CLOUD_URL)")
	cloudDevicePtr := flag.String("cloud-device", defaultCloudDeviceID, "device_id для cloud-lite (или env ODTEMP_CLOUD_DEVICE)")
	bootloaderPtr := flag.Bool("bootloader", false, "перевести устройство в загрузчик и выйти")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Использование %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Версия: %s\n", VERSION)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *bootloaderPtr {
		if err := sendBootloaderCommand(); err != nil {
			log.Fatalf("Ошибка: %v", err)
		}
		return
	}

	period, err := validatePeriod(*periodPtr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(2)
	}

	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })

	cloudWriteToken := strings.TrimSpace(*cloudTokenPtr)
	if !setFlags["cloud-token"] {
		cloudWriteToken = strings.TrimSpace(os.Getenv("ODTEMP_CLOUD_TOKEN"))
	}
	cloudURL := strings.TrimSpace(*cloudURLPtr)
	if !setFlags["cloud-url"] {
		cloudURL = envOrDefault("ODTEMP_CLOUD_URL", cloudURL)
	}
	cloudDevice := strings.TrimSpace(*cloudDevicePtr)
	if !setFlags["cloud-device"] {
		cloudDevice = envOrDefault("ODTEMP_CLOUD_DEVICE", cloudDevice)
	}
	cloudPeriodSeconds := *cloudPeriodPtr
	if !setFlags["cloud-period"] {
		cloudPeriodSeconds, err = envFloat64("ODTEMP_CLOUD_PERIOD", cloudPeriodSeconds)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
			os.Exit(2)
		}
	}

	var cloudClient *CloudClient
	var cloudPeriod time.Duration
	if cloudWriteToken != "" {
		cloudPeriod, err = validatePeriod(cloudPeriodSeconds)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка cloud-period: %v\n", err)
			os.Exit(2)
		}

		cloudClient, err = NewCloudClient(CloudConfig{
			BaseURL:    cloudURL,
			DeviceID:   cloudDevice,
			WriteToken: cloudWriteToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка cloud-настроек: %v\n", err)
			os.Exit(2)
		}
	}

	devicePeriod := deviceSamplePeriod(period, cloudPeriod, cloudClient != nil)

	guiMode := !*cliPtr && guiSupported

	if *silentPtr {
		log.SetOutput(io.Discard)
	} else {
		now := time.Now()
		logFileName := fmt.Sprintf("odtemp_%s.log", now.Format("02.01.2006_15.04.05"))

		if *pathPtr != "" {
			if err := os.MkdirAll(*pathPtr, 0755); err != nil {
				log.Fatalf("Ошибка создания директории лога: %v", err)
			}
			logFileName = filepath.Join(*pathPtr, logFileName)
		}

		logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("Ошибка открытия файла лога: %v", err)
		}
		defer logFile.Close()

		multiWriter := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(multiWriter)
	}

	log.Printf("Период записи: %.1f секунд\n", *periodPtr)
	if !*cliPtr && !guiSupported {
		log.Println("Сборка без GUI: работа в консольном режиме")
	}
	if cloudClient != nil {
		log.Printf("Cloud-lite запись включена: URL %s, device_id %s, период %.1f секунд\n", strings.TrimRight(cloudURL, "/"), cloudDevice, cloudPeriodSeconds)
		if cloudPeriod < defaultCloudTimeout {
			log.Printf("Предупреждение: cloud-period (%.1f с) меньше таймаута запроса (%.0f с); при недоступности сервера отправки могут не успевать\n", cloudPeriodSeconds, defaultCloudTimeout.Seconds())
		}
	}

	quit := make(chan struct{})
	var closeOnce sync.Once
	closeQuit := func() {
		closeOnce.Do(func() { close(quit) })
	}

	ds := &DeviceState{}

	var lastTemp float64
	var lastHumidity float64
	var lastHasHumidity bool
	var lastGen uint64
	var hasLastSample bool
	var tempMutex sync.Mutex

	sampleChans := []chan SensorSample{
		startPeriodicSampleRecorder(quit, ds, period, logSensorSample),
	}
	var wg sync.WaitGroup

	if cloudClient != nil {
		sampleChans = append(sampleChans, startPeriodicSampleRecorder(quit, ds, cloudPeriod, func(sample SensorSample) {
			if err := cloudClient.PostSampleWithTimeout(sample); err != nil {
				log.Printf("Ошибка записи в cloud-lite: %v\n", err)
			}
		}))
	}

	var ui *UI
	if guiMode {
		var err error
		ui, err = NewUI()
		if err != nil {
			log.Printf("Ошибка создания UI: %v, переключение в CLI режим", err)
			showSystemDialog(
				"Монитор ODTEMP-1",
				"Не удалось запустить графический интерфейс.\nПриложение продолжит работу в консольном режиме.\n\nДля выхода нажмите Ctrl+C в терминале.",
			)
			guiMode = false
		} else {
			ui.SetOnClosed(func() {
				closeQuit()
			})
		}
	}

	if guiMode && ui != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()

			uiTicker := time.NewTicker(200 * time.Millisecond)
			defer uiTicker.Stop()

			for {
				select {
				case <-uiTicker.C:
					if _, found := ds.getDevice(); found {
						tempMutex.Lock()
						t := lastTemp
						h := lastHumidity
						hasHumidity := lastHasHumidity
						gen := lastGen
						hasSample := hasLastSample
						tempMutex.Unlock()
						if hasSample && gen == ds.getGeneration() {
							ui.UpdateMeasurements(t, h, hasHumidity)
						} else {
							ui.ShowWaiting()
						}
					}
				case <-quit:
					return
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		if !searchDevice(ds, quit, *silentPtr) {
			return
		}

		log.Println("Запуск цикла чтения данных с устройства")
		time.Sleep(500 * time.Millisecond)
		configureFastDeviceInterval(ds, devicePeriod.Seconds())

		replyTimeout := missedReadLimit
		report := make([]byte, 64)

		reconnect := func(showStatus func()) bool {
			if showStatus != nil {
				showStatus()
			}
			if !searchDevice(ds, quit, *silentPtr) {
				return false
			}
			time.Sleep(500 * time.Millisecond)
			configureFastDeviceInterval(ds, devicePeriod.Seconds())
			replyTimeout = missedReadLimit
			return true
		}

		for {
			select {
			case <-quit:
				return
			default:
				dev, found := ds.getDevice()
				if !found || dev == nil {
					time.Sleep(100 * time.Millisecond)
					continue
				}

				n, err := dev.ReadWithTimeout(report, readTimeout)
				if errors.Is(err, hid.ErrTimeout) {
					n = 0
				} else if err != nil {
					log.Printf("Ошибка чтения: %v\n", err)
					ds.clearDevice()

					var showStatus func()
					if guiMode && ui != nil {
						showStatus = ui.ShowDisconnected
					}
					if reconnect(showStatus) {
						continue
					}
					return
				}

				if n > 0 {
					data := report[:n]
					reportID := data[0]

					switch reportID {
					case HID_DATA_REPORT_ID:
						if len(data) < 3 {
							continue
						}
						temp, humidity, hasHumidity, ok := processDataReport(data[1:])
						if !ok {
							continue
						}

						nowSample := time.Now()
						tempMutex.Lock()
						lastTemp = temp
						lastHumidity = humidity
						lastHasHumidity = hasHumidity
						lastGen = ds.getGeneration()
						hasLastSample = true
						tempMutex.Unlock()

						sample := SensorSample{
							Temperature: temp,
							Humidity:    humidity,
							HasHumidity: hasHumidity,
							At:          nowSample,
							Gen:         ds.getGeneration(),
						}
						for _, sampleChan := range sampleChans {
							offerSample(sampleChan, sample)
						}

					case HID_EVENT_REPORT_ID:

					case HID_FW_REPORT_ID:
						if len(data) > 2 {
							length := int(data[1])
							if 2+length <= len(data) {
								firmwareVersion := string(data[2 : 2+length])
								log.Printf("[FW] Версия прошивки: %s\n", firmwareVersion)
							}
						}

					case HID_CMD_REPORT_ID:
						if len(data) > 1 {
							cmd := data[1]
							if cmd == HID_CMD_RST_DFU || cmd == HID_CMD_RST_UAPP || cmd == HID_CMD_RST_STM {
								log.Println("Устройство переходит в режим сброса/DFU. Закрытие устройства.")
								ds.clearDevice()

								if !guiMode {
									closeQuit()
									return
								}
								if reconnect(ui.ShowDisconnected) {
									continue
								}
								return
							}
							log.Printf("Получена команда 0x%X с данными: %X\n", cmd, data[2:])
						}

					default:
						log.Printf("Неизвестный report id: %d\n", reportID)
					}

					replyTimeout = missedReadLimit
				} else if replyTimeout > 0 {
					replyTimeout--
				} else {
					log.Println("Устройство не отвечает – превышен таймаут")
					ds.clearDevice()

					var showStatus func()
					if guiMode && ui != nil {
						showStatus = ui.ShowConnectionLost
					}
					if reconnect(showStatus) {
						continue
					}
					return
				}
			}
		}
	}()

	if guiMode && ui != nil {
		ui.Run()
	} else {
		<-quit
	}

	closeQuit()
	wg.Wait()

	ds.clearDevice()
	hid.Exit()
	log.Println("Приложение закрыто")
}
