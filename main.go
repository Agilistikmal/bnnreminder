package main

import (
	"log"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type KGBData struct {
	No            string
	Nama          string
	NIP           string
	Pangkat       string
	Gol           string
	TempatLahir   string
	TanggalLahir  string
	TMTLama       string
	GajiPokokLama string
	MasaKerjaLama string
	TMTBaru       string
	GajiPokokBaru string
	MasaKerjaBaru string
	KGBBerikutnya string
	OlehPejabat   string
	NomorSRT      string
	TanggalSRT    string
	Tembusan      string
	Tembusan1     string
	Satker        string
	Di            string
}

func main() {
	log.Println("Starting...")

	log.Println("Loading Excel file...")
	f, err := excelize.OpenFile("data.xlsx")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Loading Session Database...")
	db, err := gorm.Open(sqlite.Open("session.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	err = db.AutoMigrate()
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	rows, err := f.GetRows("KGB")
	if err != nil {
		log.Fatal(err)
	}

	if len(rows) == 0 {
		log.Fatal("Sheet kosong")
	}

	headerIndex := make(map[string]int)
	for i, col := range rows[0] {
		headerIndex[col] = i
	}

	var dataList []KGBData
	for _, row := range rows[1:] {
		if row[0] == "" {
			continue
		}

		get := func(header string) string {
			if idx, ok := headerIndex[header]; ok && idx < len(row) {
				return row[idx]
			}
			return ""
		}

		data := KGBData{
			No:            get("NO"),
			Nama:          get("NAMA"),
			NIP:           get("NIP"),
			Pangkat:       get("PANGKAT"),
			Gol:           get("GOL"),
			TempatLahir:   get("TMP LAHIR"),
			TanggalLahir:  get("TGL LAHIR"),
			TMTLama:       get("TMT KGB LAMA/\nPANGKAT"),
			GajiPokokLama: get("GAJI POKOK LAMA"),
			MasaKerjaLama: get("MASA KERJA LAMA"),
			TMTBaru:       get("TMT KGB  BARU"),
			GajiPokokBaru: get("GAJI POKOK BARU"),
			MasaKerjaBaru: get("MASA KERJA BARU"),
			KGBBerikutnya: get("KGB BERIKUTNYA"),
			OlehPejabat:   get("OLEH PEJABAT"),
			NomorSRT:      get("NOMOR_SRT"),
			TanggalSRT:    get("TGL"),
			Tembusan:      get("TEMBUSAN"),
			Tembusan1:     get("TEMBUSAN_1"),
			Satker:        get("Satker"),
			Di:            get("di"),
		}

		dataList = append(dataList, data)
	}

	for _, d := range dataList {
		if d.TMTLama == "" {
			log.Printf("Data TMT Lama tidak ada untuk No %s (NIP: %s)\n", d.No, d.NIP)
			continue
		}
		tmtLama, err := time.Parse("02-01-2006", d.TMTLama)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		tmtBaru := tmtLama.AddDate(2, 0, 0)
		now := time.Now()
		twoMonthsLater := now.AddDate(0, 2, 0)

		// Cek apakah TMT Baru berada di antara sekarang dan dua bulan ke depan
		if tmtBaru.After(now) && tmtBaru.Before(twoMonthsLater) {
			log.Println("-")
			log.Println("Now       :", now.Format("02-01-2006"))
			log.Println("TMT Lama  :", tmtLama.Format("02-01-2006"))
			log.Println("TMT Baru  :", tmtBaru.Format("02-01-2006"))
			log.Println("-")
		}
	}
}

func SendWhatsAppNotification(data *KGBData) error {
	// Implementasi pengiriman pesan WhatsApp
	// Misalnya menggunakan Twilio, WhatsApp API, atau library lain
	return nil
}
