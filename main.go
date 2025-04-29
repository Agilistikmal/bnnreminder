package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/xuri/excelize/v2"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
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

type Notified struct {
	NIP     string `gorm:"primaryKey,column:nip"`
	TMTLama string
	TMTBaru string
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

	err = db.AutoMigrate(&Notified{})
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	// Whatsmeow
	container, err := sqlstore.New("sqlite3", "file:session.db?_foreign_keys=on", nil)
	if err != nil {
		log.Fatalf("gagal buat store container: %v", err)
	}

	// Ambil device pertama dari DB
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		log.Fatalf("gagal ambil device: %v", err)
	}

	client := whatsmeow.NewClient(deviceStore, nil)

	// Cek sudah login atau belum
	if client.Store.ID == nil {
		// Belum login, scan QR code
		qrChan, _ := client.GetQRChannel(context.Background())
		err := client.Connect()
		if err != nil {
			log.Fatalf("gagal connect: %v", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Scan QR ini untuk login WhatsApp")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login status:", evt.Event)
			}
		}
	} else {
		// Sudah login, langsung connect
		err := client.Connect()
		if err != nil {
			log.Fatalf("gagal reconnect: %v", err)
		}
	}

	// Ambil data dari sheet "KGB"
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
			// Cek apakah sudah diberitahu sebelumnya
			var notified Notified
			err = db.First(&notified, "nip = ? AND tmt_lama = ? AND tmt_baru = ?", d.NIP, tmtLama, tmtBaru).Error
			if err == nil {
				log.Printf("Sudah diberitahu sebelumnya untuk No %s (NIP: %s)\n", d.No, d.NIP)
				continue
			}
			// Simpan data yang sudah diberitahu
			notified = Notified{
				NIP:     d.NIP,
				TMTLama: tmtLama.Format("02-01-2006"),
				TMTBaru: tmtBaru.Format("02-01-2006"),
			}
			err = db.Create(&notified).Error
			if err != nil {
				log.Printf("Gagal simpan data yang sudah diberitahu: %v\n", err)
				continue
			}
			// Kirim notifikasi WhatsApp
			err = SendWhatsAppNotification(client, &d, &now, &tmtLama, &tmtBaru)
			if err != nil {
				log.Printf("Gagal kirim notifikasi WhatsApp: %v\n", err)
				continue
			}
			log.Printf("Notifikasi terkirim untuk No %s (NIP: %s)\n", d.No, d.NIP)
		}
	}

	// Tunggu sampai app dimatikan
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("Menutup koneksi...")
	client.Disconnect()
}

func SendWhatsAppNotification(client *whatsmeow.Client, data *KGBData, now *time.Time, tmtLama *time.Time, tmtBaru *time.Time) error {
	// Kirim notifikasi WhatsApp ke grup
	// Ganti dengan nomor grup WhatsApp yang sesuai
	groupJID := types.NewJID("120363399863476722", "g.us")

	// Kirim pesan ke grup
	text := fmt.Sprintf(
		`
*Notifikasi KGB*
No: *%s*

Nama: %s
NIP: *%s*

TMT Lama: *%s*
TMT Baru: *%s*
		`, data.No, data.Nama, data.NIP, tmtLama.Format("02-01-2006"), tmtBaru.Format("02-01-2006"))
	_, err := client.SendMessage(context.Background(), groupJID, &waE2E.Message{
		Conversation: &text,
	})
	if err != nil {
		return fmt.Errorf("gagal kirim pesan ke grup: %w", err)
	}
	return nil
}
