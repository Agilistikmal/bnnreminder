package main

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
