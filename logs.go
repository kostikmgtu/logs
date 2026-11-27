package logs

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var logsFile *os.File
var catalog string

// эта функция выполняется многократно, изменяя название файла логов в зависимости от текущей даты
func checkLogs() {
	currentDir, err := os.Getwd()
	if err != nil {
		logMessage("Ошибка при получении текущего каталога: " + err.Error())
		return
	}

	// Формируем путь к подкаталогу
	logsDir := filepath.Join(currentDir, catalog)

	// Проверяем, существует ли каталог
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		err = os.MkdirAll(logsDir, 0755)
		if err != nil {
			logMessage("Ошибка при создании каталога " + catalog + ": " + err.Error())
			return
		}
		logMessage("Каталог успешно создан по пути: " + logsDir)
	} else if err != nil {
		logMessage("Ошибка при проверке каталога " + catalog + ": " + err.Error())
		return
	}

	currentTimeFormat := time.Now().Format("2006-01-02")
	currFileName := filepath.Join(logsDir, "/logs-"+currentTimeFormat)

	// проверяем, не идет ли уже логирование в этот же файл
	if logsFile != nil {
		if logsFile.Name() == currFileName {
			return
		}
		// Закрываем предыдущий файл перед переключателем
		logsFile.Close()
	}

	// Открываем или создаем новый файл
	newFile, err := os.OpenFile(currFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logMessage("Ошибка открытия файла логов '" + currFileName + "': " + err.Error())
		return
	}

	logsFile = newFile

	// Перенаправляем stdout, stderr и стандартный пакета log в новый файл
	os.Stdout = logsFile
	os.Stderr = logsFile
	log.SetOutput(logsFile)
}

// точка входа для приложений.
func InitFileLogs(logsCatalog string, days int, fileServerPort string) {
	catalog = logsCatalog

	// создаем или прицепляемся к нужному файлу вывода
	checkLogs()

	// запускаем файл-сервер с логами
	if fileServerPort != "" {
		go runFileServerLogs(fileServerPort)
	}

	scheduleDailyLogRotation(days)
}

// ежедневно выполняемые действия по обслуживанию приложения
func scheduleDailyLogRotation(days int) {
	go func() {
		for {
			now := time.Now()

			nextMidnight := time.Date(
				now.Year(), now.Month(), now.Day()+1,
				0, 0, 0, 0,
				now.Location(),
			)

			duration := nextMidnight.Sub(now)
			time.Sleep(duration)

			checkLogs()
			deleteOldLogs(days)
		}
	}()
}

// удаление логов старше указанного количества дней
func deleteOldLogs(days int) {
	currentDir, err := os.Getwd()
	if err != nil {
		logMessage("Ошибка получения текущего каталога: " + err.Error())
		return
	}

	logsDir := filepath.Join(currentDir, catalog)

	files, err := os.ReadDir(logsDir)
	if err != nil {
		logMessage("Ошибка чтения каталога " + catalog + ": " + err.Error())
		return
	}

	expireDuration := time.Duration(days) * 24 * time.Hour
	now := time.Now()

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(logsDir, file.Name())

		info, err := file.Info()
		if err != nil {
			logMessage("Ошибка получения информации о файле " + file.Name() + ": " + err.Error())
			continue
		}

		if now.Sub(info.ModTime()) > expireDuration {
			err := os.Remove(filePath)
			if err != nil {
				logMessage("Ошибка удаления файла " + file.Name() + ": " + err.Error())
			} else {
				logMessage("Удалён старый лог: " + file.Name())
			}
		}
	}
}

func runFileServerLogs(port string) {
	logsDir := filepath.Join(".", catalog)
	fileServer := http.FileServer(http.Dir(logsDir))
	http.Handle("/logs/", http.StripPrefix("/logs/", fileServer))

	logMessage("Сервер логирования запущен на http://localhost:" + port + "/logs/")
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		//fmt.Fprintf(os.Stderr, "Ошибка запуска HTTP сервера логов: %v\n", err)
		logMessage("Ошибка запуска HTTP сервера логов: " + err.Error())
	}
}

func logMessage(msg string) {
	t := time.Now().Format("2006-01-02 15:04:05") + ". "
	fmt.Printf("%s%s\n", t, msg)
}
