package main

import (
	"encoding/csv"
	"io"
	"log"
	"net/http"
)

func ReadGoogleSpreadsheet(url string) []KGBData {
	log.Println("Downloading Google Spreadsheet as CSV...")
	resp, err := http.Get("https://docs.google.com/spreadsheets/d/1aUaVK6m6NMsw0hliH-wwlqb2ayLd6CHuT8F0rIUNvyM/export?format=csv&gid=0")
	if err != nil {
		log.Fatalf("Gagal mengunduh spreadsheet: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Gagal mengunduh file: status %d", resp.StatusCode)
	}

	// Baca seluruh konten sebagai []byte
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Gagal membaca body: %v", err)
	}

	// Perbaiki karakter bermasalah: hapus/mask kutip liar
	cleaned := make([]byte, 0, len(body))
	inQuotes := false
	for i := 0; i < len(body); i++ {
		b := body[i]
		if b == '"' {
			// Coba deteksi kutip liar
			if !inQuotes && (i == 0 || body[i-1] == ',' || body[i-1] == '\n') {
				inQuotes = true
			} else if inQuotes && (i+1 == len(body) || body[i+1] == ',' || body[i+1] == '\n') {
				inQuotes = false
			} else {
				// kutip liar, ganti jadi spasi atau skip
				continue
			}
		}
		cleaned = append(cleaned, b)
	}

	// Parse CSV dari hasil cleaning
	reader := csv.NewReader(stringReader(string(cleaned)))
	reader.FieldsPerRecord = -1 // biar fleksibel jumlah kolom
	rows, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Gagal baca CSV setelah dibersihkan: %v", err)
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
			TMTLama:       get("TMT KGB LAMA/PANGKAT"),
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

	log.Printf("Berhasil membaca %d data dari spreadsheet\n", len(dataList))

	return dataList
}
