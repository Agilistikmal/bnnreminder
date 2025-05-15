package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
)

func NewWhatsAppClient() *whatsmeow.Client {
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

	return client
}

func SendWhatsAppNotification(client *whatsmeow.Client, data *KGBData, now *time.Time, tmtLama *time.Time, tmtBaru *time.Time) error {
	// Kirim notifikasi WhatsApp ke grup
	// Ganti dengan nomor grup WhatsApp yang sesuai
	groupJID := types.NewJID("120363399863476722", "g.us")

	// Format tanggal Indonesia
	tmtLamaStr := tmtLama.Format("02 January 2006")
	tmtBaruStr := tmtBaru.Format("02 January 2006")

	// Kirim pesan ke grup
	text := fmt.Sprintf(
		`🔔 *NOTIFIKASI KENAIKAN GAJI BERKALA* 🔔
━━━━━━━━━━━━━━━━━━━━━

👤 *INFORMASI PEGAWAI*
📝 Nomor: %s
👨‍💼 Nama: *%s*
🆔 NIP: *%s*
📊 Pangkat/Gol: *%s/%s*

📅 *INFORMASI KGB*
• TMT Lama: *%s*
• Gaji Pokok Lama: *Rp%s*
• Masa Kerja Lama: *%s*

📈 *KENAIKAN GAJI BERKALA*
• TMT Baru: *%s*
• Gaji Pokok Baru: *Rp%s*
• Masa Kerja Baru: *%s*

📋 *INFORMASI SURAT*
• Nomor: *%s*
• Tanggal: *%s*
• Pejabat: *%s*

🏢 *UNIT KERJA*
• Satker: *%s*
• Lokasi: *%s*

⚠️ _Mohon segera mempersiapkan berkas-berkas yang diperlukan._
⚠️ _Jangan lupa untuk memperbarui data pegawai di https://s.id/D3gqN juga_
━━━━━━━━━━━━━━━━━━━━━
		`,
		data.No,
		data.Nama,
		data.NIP,
		data.Pangkat, data.Gol,
		tmtLamaStr,
		data.GajiPokokLama,
		data.MasaKerjaLama,
		tmtBaruStr,
		data.GajiPokokBaru,
		data.MasaKerjaBaru,
		data.NomorSRT,
		data.TanggalSRT,
		data.OlehPejabat,
		data.Satker,
		data.Di,
	)

	_, err := client.SendMessage(context.Background(), groupJID, &waE2E.Message{
		Conversation: &text,
	})
	if err != nil {
		return fmt.Errorf("gagal kirim pesan ke grup: %w", err)
	}
	return nil
}

func stringReader(s string) io.Reader {
	return &stringReaderImpl{s: s}
}

type stringReaderImpl struct {
	s string
	i int64
}

func (r *stringReaderImpl) Read(p []byte) (int, error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.i:])
	r.i += int64(n)
	return n, nil
}
