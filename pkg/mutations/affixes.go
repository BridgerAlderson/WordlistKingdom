package mutations

var commonPrefixes = []string{
	"Admin", "admin",
	"Test", "test",
	"User", "user",
	"Dev", "dev",
	"IT", "it",
	"Sistem", "sistem",
	"Helpdesk", "helpdesk",
	"Root", "root",
	"Super", "super",
	"Backup", "backup",
	"Net", "net",
	"Sec", "sec",
}

var commonSuffixes = []string{
	"Admin", "admin",
	"Test", "test",
	"User", "user",
	"Dev", "dev",
	"IT", "it",
	"123", "1234", "12345",
	"0", "01", "99", "00",
}

func sendAffixVariants(kw string, out chan<- string) {
	for _, p := range commonPrefixes {
		out <- p + kw
		out <- p + "_" + kw
		out <- p + "." + kw
	}
	for _, sf := range commonSuffixes {
		out <- kw + sf
		out <- kw + "_" + sf
		out <- kw + "." + sf
	}
}
