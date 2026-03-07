package logs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var logsFile *os.File
var catalog string

func checkLogs() {
	currentDir, err := os.Getwd()

	if err != nil {
		fmt.Fprintf(os.Stdout, "Ошибка при получении текущего каталога: %v\n", err)
		return
	}

	// Формируем путь к подкаталогу
	logsDir := filepath.Join(currentDir, catalog)

	// Проверяем, существует ли каталог
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		// Если не существует — создаём
		err = os.MkdirAll(logsDir, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка при создании каталога %s: %v\n", catalog, err)
			return
		}
		fmt.Printf("Каталог успешно создан по пути: %s\n", logsDir)
	} else if err != nil {
		// Если возникла другая ошибка
		fmt.Fprintf(os.Stderr, "Ошибка при проверке каталога %s: %v\n", catalog, err)
		return
	} else {
		//fmt.Printf("Каталог 'logs' уже существует по пути: %s\n", logsDir)
	}

	currentTimeFormat := time.Now().Format("2006-01-02")
	currFileName := logsDir + "/logs-" + currentTimeFormat

	//проверяем, не идет ли уже логирование, так как процедуру инициации логов мы проводим при начале каждой дедубликации
	if logsFile != nil {
		if logsFile.Name() == currFileName {
			return
		}
	}

	_, err = os.Stat(currFileName)
	if os.IsNotExist(err) {
		//значит файл с логами надо создать с новым именем
		logsFile, err = os.Create(currFileName)
		if err != nil {
			log.Printf("Не удалось создать файл логов: %s\n", err.Error())
		}
	} else {
		logsFile, err = os.OpenFile(currFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Ошибка открытия файла логов '%s': %v\n", currFileName, err)
			return
		}
	}

	os.Stdout = logsFile
}

func InitFileLogs(logsCatalog string, days int) {

	catalog = logsCatalog
	checkLogs()

	scheduleDailyLogRotation(days)
}

// ежедневно выполняемые действия по обслуживанию приложения
func scheduleDailyLogRotation(days int) {

	go func() {
		for {
			// текущее время
			now := time.Now()

			// ближайшая следующая полночь
			nextMidnight := time.Date(
				now.Year(), now.Month(), now.Day()+1,
				0, 0, 0, 0,
				now.Location(),
			)

			// длительность ожидания до следующей полуночи
			duration := nextMidnight.Sub(now)

			// ждём
			time.Sleep(duration)

			// вызываем вашу функцию
			checkLogs()

			deleteOldLogs(days)
		}
	}()
}

// удаление логов старше указанного количества дней
func deleteOldLogs(days int) {
	currentDir, err := os.Getwd()
	if err != nil {
		log.Printf("Ошибка получения текущего каталога: %v\n", err)
		return
	}

	logsDir := filepath.Join(currentDir, "logs")

	files, err := os.ReadDir(logsDir)
	if err != nil {
		log.Printf("Ошибка чтения каталога logs: %v\n", err)
		return
	}

	expireDuration := time.Duration(days) * 24 * time.Hour
	now := time.Now()

	for _, file := range files {

		// пропускаем директории
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(logsDir, file.Name())

		info, err := file.Info()
		if err != nil {
			log.Printf("Ошибка получения информации о файле %s: %v\n", file.Name(), err)
			continue
		}

		// если файл старше указанного срока
		if now.Sub(info.ModTime()) > expireDuration {

			err := os.Remove(filePath)
			if err != nil {
				log.Printf("Ошибка удаления файла %s: %v\n", file.Name(), err)
			} else {
				log.Printf("Удалён старый лог: %s\n", file.Name())
			}
		}
	}
}
