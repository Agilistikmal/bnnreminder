package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("Starting...")

	log.Println("Loading Session Database...")
	db := NewDatabase()

	log.Println("Loading WhatsApp Client...")
	client := NewWhatsAppClient()

	var url string
	if _, err := os.Stat("url.txt"); os.IsNotExist(err) {
		url := "https://docs.google.com/spreadsheets/d/1aUaVK6m6NMsw0hliH-wwlqb2ayLd6CHuT8F0rIUNvyM/edit?hl=id&gid=0#gid=0"
		if err := os.WriteFile("url.txt", []byte(url), 0644); err != nil {
			log.Fatalf("Gagal menulis ke file url.txt: %v", err)
		}
		log.Println("File url.txt dibuat dengan URL default.")
	}

	file, err := os.Open("url.txt")
	if err != nil {
		log.Fatalf("Gagal membuka file url.txt: %v", err)
	}
	defer file.Close()

	_, err = fmt.Fscanf(file, "%s", &url)
	if err != nil {
		log.Fatalf("Gagal membaca URL dari file url.txt: %v", err)
	}

	dataList := ReadGoogleSpreadsheet(url)

	log.Println("Melakukan iterasi data...")
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
