package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/robfig/cron/v3"
)

func main() {
	fmt.Println("")
	figure.NewColorFigure("Notifikasi KGB", "doom", "cyan", true).Print()
	fmt.Println("")
	fmt.Println("Informatika UTY - Agil GI")
	fmt.Println("")

	log.Println("Starting...")

	log.Println("Loading Session Database...")
	db := NewDatabase()

	log.Println("Loading WhatsApp Client...")
	client := NewWhatsAppClient()

	log.Println("Creating Cron Job...")
	loc, _ := time.LoadLocation("Asia/Jakarta")
	cron := cron.New(cron.WithLocation(loc))

	defaultUrl := "aHR0cHM6Ly9kb2NzLmdvb2dsZS5jb20vc3ByZWFkc2hlZXRzL2QvMWFVYVZLNm02Tk1zdzBobGlILXd3bHFiMmF5TGQ2Q0h1VDhGMHJJVU52eU0vZWRpdD9obD1pZCZnaWQ9MCNnaWQ9MA=="

	// Every 1 minute
	log.Println("Cron Job Running... (setiap 1 menit)")
	cron.AddFunc("* * * * *", func() {
		// Clear console
		fmt.Print("\033c")
		fmt.Println("")
		figure.NewColorFigure("Notifikasi KGB", "doom", "cyan", true).Print()
		fmt.Println("")
		fmt.Println("== Waktu pengecekan terakhir:", time.Now().Format("15:04:05"))
		var url string
		if _, err := os.Stat("url.txt"); os.IsNotExist(err) {
			url, _ := base64.StdEncoding.DecodeString(defaultUrl)
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

			// Konversi ke WIB
			loc, _ := time.LoadLocation("Asia/Jakarta")
			tmtLama = tmtLama.In(loc)
			tmtBaru := tmtLama.AddDate(2, 0, 0)
			now := time.Now().In(loc)

			// Hitung 2 bulan sebelum TMT Baru
			twoMonthsBeforeTMTBaru := tmtBaru.AddDate(0, -2, 0)

			// Cek apakah sekarang sudah masuk periode 2 bulan sebelum TMT Baru
			if now.After(twoMonthsBeforeTMTBaru) && now.Before(tmtBaru) {
				// Cek apakah sudah diberitahu sebelumnya
				var notified Notified
				err = db.First(&notified, "nip = ? AND tmt_lama = ? AND tmt_baru = ?", d.NIP, tmtLama.Format("02-01-2006"), tmtBaru.Format("02-01-2006")).Error
				if err == nil {
					log.Printf("Sudah diberitahu sebelumnya untuk No %s (NIP: %s), namun belum ada perubahan TMT Baru\n", d.No, d.NIP)
					continue
				}
				// Simpan data yang sudah diberitahu
				notified = Notified{
					NIP:     d.NIP,
					TMTLama: tmtLama.Format("02-01-2006"),
					TMTBaru: tmtBaru.Format("02-01-2006"),
				}
				err = db.Save(&notified).Error
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
	})

	cron.Start()

	// Tunggu sampai app dimatikan
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("Menutup koneksi...")
	client.Disconnect()
}
