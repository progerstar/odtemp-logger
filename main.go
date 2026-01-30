package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sstallion/go-hid"
)

// Константы устройства
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
	VERSION          = "1.3.0"
)

// TempSample хранит отсчёт температуры
type TempSample struct {
	Value float64
	At    time.Time
	Gen   uint64
}

// DeviceState хранит состояние устройства
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

// sendBootloaderCommand отправляет команду перехода в загрузчик и сразу закрывает устройство
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

// findAndOpenDevice выполняет поиск и открытие первого доступного устройства
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

// processDataReport обрабатывает отчёт с данными
func processDataReport(data []byte) float64 {
	if len(data) >= 2 {
		raw := int16(binary.LittleEndian.Uint16(data[:2]))
		return float64(raw) / 100.0
	}
	return 0
}

// searchDevice ищет устройство с переподключением
func searchDevice(ds *DeviceState, quit <-chan struct{}, silent bool) <-chan struct{} {
	foundChan := make(chan struct{})

	go func() {
		for {
			dev, err := findAndOpenDevice()
			if err == nil {
				ds.setDevice(dev)
				log.Println("Устройство успешно открыто")
				close(foundChan)
				return
			}

			if !silent {
				log.Println(err)
				log.Println("Повторная попытка через 1 сек...")
			}

			select {
			case <-quit:
				return
			case <-time.After(1 * time.Second):
			}
		}
	}()

	return foundChan
}

func main() {
	// Парсинг параметров командной строки
	cliPtr := flag.Bool("cli", false, "запуск без GUI")
	pathPtr := flag.String("path", "", "переопределяет путь записи лога")
	silentPtr := flag.Bool("silent", false, "не писать лог")
	periodPtr := flag.Float64("period", 60, "период записи в секундах")
	bootloaderPtr := flag.Bool("bootloader", false, "перевести устройство в загрузчик и выйти")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Использование %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Версия: %s\n", VERSION)
		flag.PrintDefaults()
	}
	flag.Parse()

	// Режим bootloader - отправить команду и выйти
	if *bootloaderPtr {
		if err := sendBootloaderCommand(); err != nil {
			log.Fatalf("Ошибка: %v", err)
		}
		return
	}

	// Режим работы (GUI или CLI)
	guiMode := !*cliPtr

	// Настройка логирования
	if !*silentPtr {
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

	// Каналы управления
	quit := make(chan struct{})
	var closeOnce sync.Once
	closeQuit := func() {
		closeOnce.Do(func() { close(quit) })
	}

	// Состояние устройства
	ds := &DeviceState{}

	// Температура
	var lastTemp float64
	var tempMutex sync.Mutex

	// Периодическое логирование
	var sampleChan chan TempSample
	var logPeriod time.Duration

	if *periodPtr >= 2 {
		logPeriod = time.Duration(*periodPtr * float64(time.Second))
		sampleChan = make(chan TempSample, 1)

		go func() {
			now := time.Now()
			firstTick := now.Truncate(logPeriod).Add(logPeriod)
			timer := time.NewTimer(time.Until(firstTick))
			defer timer.Stop()

			bySlot := make(map[int64]TempSample)

			for {
				select {
				case <-quit:
					return
				case s := <-sampleChan:
					if ds.isAlive() && s.Gen == ds.getGeneration() {
						slotEnd := s.At.Truncate(logPeriod).Add(logPeriod)
						bySlot[slotEnd.UnixNano()] = s
					}
				case tickTime := <-timer.C:
					prevEndUnix := tickTime.Truncate(logPeriod).UnixNano()
					if s, ok := bySlot[prevEndUnix]; ok {
						if ds.isAlive() && s.Gen == ds.getGeneration() {
							log.Printf("Температура: %.2f°C\n", s.Value)
						}
						delete(bySlot, prevEndUnix)
					}
					next := tickTime.Add(logPeriod)
					timer.Reset(time.Until(next))
				}
			}
		}()
	}

	// Инициализация UI (только в GUI режиме)
	var ui *UI
	if guiMode {
		var err error
		ui, err = NewUI()
		if err != nil {
			log.Printf("Ошибка создания UI: %v, переключение в CLI режим", err)
			showSystemDialog(
				"Монитор температуры",
				"Не удалось запустить графический интерфейс.\nПриложение продолжит работу в консольном режиме.\n\nДля выхода нажмите Ctrl+C в терминале.",
			)
			guiMode = false
		} else {
			ui.SetOnClosed(func() {
				closeQuit()
			})
		}
	}

	// Поиск устройства
	deviceFoundChan := searchDevice(ds, quit, *silentPtr)

	// Горутина обновления UI
	if guiMode && ui != nil {
		go func() {
			uiTicker := time.NewTicker(200 * time.Millisecond)
			defer uiTicker.Stop()

			for {
				select {
				case <-uiTicker.C:
					if _, found := ds.getDevice(); found {
						tempMutex.Lock()
						t := lastTemp
						tempMutex.Unlock()
						ui.UpdateTemperature(t)
					}
				case <-quit:
					return
				}
			}
		}()
	}

	// Горутина чтения данных
	go func() {
		select {
		case <-deviceFoundChan:
			log.Println("Запуск цикла чтения данных с устройства")
			time.Sleep(500 * time.Millisecond)

			// Настройка интервала устройства при быстром периоде
			if *periodPtr < 2 {
				dev, _ := ds.getDevice()
				if dev != nil {
					var intervalMs uint32
					if *periodPtr <= 0.2 {
						intervalMs = 200
					} else {
						intervalMs = uint32(*periodPtr * 1000)
					}

					if err := setDeviceInterval(dev, intervalMs); err != nil {
						log.Println(err)
					}
					if interval, err := getDeviceInterval(dev); err == nil {
						log.Printf("Полученный интервал: %d ms\n", interval)
					}
				}
			}

			replyTimeout := 9
			report := make([]byte, 64)

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

					n, err := dev.Read(report)
					if err != nil {
						log.Printf("Ошибка чтения: %v\n", err)
						ds.clearDevice()

						if guiMode && ui != nil {
							ui.ShowDisconnected()
							deviceFoundChan = searchDevice(ds, quit, *silentPtr)

							select {
							case <-deviceFoundChan:
								time.Sleep(500 * time.Millisecond)
								continue
							case <-quit:
								return
							}
						} else {
							closeQuit()
							return
						}
					}

					if n > 0 {
						data := report[:n]
						reportID := data[0]

						switch reportID {
						case HID_DATA_REPORT_ID:
							if len(data) < 3 {
								continue
							}
							temp := processDataReport(data[1:])

							nowSample := time.Now()
							tempMutex.Lock()
							lastTemp = temp
							tempMutex.Unlock()

							if sampleChan != nil {
								select {
								case sampleChan <- TempSample{Value: temp, At: nowSample, Gen: ds.getGeneration()}:
								default:
									select {
									case <-sampleChan:
									default:
									}
									select {
									case sampleChan <- TempSample{Value: temp, At: nowSample, Gen: ds.getGeneration()}:
									default:
									}
								}
							}

							if *periodPtr < 2 {
								log.Printf("Температура: %.2f°C\n", temp)
							}

						case HID_EVENT_REPORT_ID:
							// Событие сенсора - обрабатывается молча

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
								} else {
									log.Printf("Получена команда 0x%X с данными: %X\n", cmd, data[2:])
								}
							}

						default:
							log.Printf("Неизвестный report id: %d\n", reportID)
						}

						replyTimeout = 9
					} else {
						if replyTimeout > 0 {
							replyTimeout--
						} else {
							log.Println("Устройство не отвечает – превышен таймаут")
							ds.clearDevice()

							if guiMode && ui != nil {
								ui.ShowConnectionLost()
								deviceFoundChan = searchDevice(ds, quit, *silentPtr)

								select {
								case <-deviceFoundChan:
									time.Sleep(500 * time.Millisecond)
									replyTimeout = 9
									continue
								case <-quit:
									return
								}
							} else {
								closeQuit()
								return
							}
						}
					}
				}
			}
		case <-quit:
			return
		}
	}()

	// Основной цикл
	if guiMode && ui != nil {
		ui.Run()
	} else {
		<-quit
	}

	// Очистка
	ds.clearDevice()
	hid.Exit()
	log.Println("Приложение закрыто")
}
