package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/0xhandle/revenger-clientless/internal/login"
	"github.com/0xhandle/revenger-clientless/internal/serverlist"
)

const (
	defaultHost = "82.38.10.101"
	defaultPort = 19101
	deviceSalt  = "revenger-clientless/"
)

type options struct {
	host       string
	port       int
	username   string
	password   string
	pattern    string
	device     string
	jsonOutput bool
	timeout    time.Duration
}

func main() {
	opts := parseFlags()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, opts); err != nil {
		fmt.Fprintln(os.Stderr, "hata:", err)
		os.Exit(1)
	}
}

func parseFlags() options {
	var o options
	flag.StringVar(&o.username, "user", "", "kullanıcı adı (zorunlu)")
	flag.StringVar(&o.password, "password", "", "şifre (zorunlu)")
	flag.StringVar(&o.pattern, "pattern", "", "yeni cihazda istenen desen, örn. 1,2,3,6,9")
	flag.StringVar(&o.device, "device", "", "32 karakterlik cihaz kimliği (boş bırakılırsa kullanıcı adından üretilir)")
	flag.StringVar(&o.host, "host", defaultHost, "login sunucusu adresi")
	flag.IntVar(&o.port, "port", defaultPort, "login sunucusu portu")
	flag.BoolVar(&o.jsonOutput, "json", false, "sunucu listesini json olarak yazdır")
	flag.DurationVar(&o.timeout, "timeout", 10*time.Second, "bağlantı zaman aşımı")
	flag.Parse()
	return o
}

func run(ctx context.Context, o options) error {
	if o.username == "" || o.password == "" {
		flag.Usage()
		return errors.New("-user ve -password parametreleri zorunlu")
	}
	if o.device == "" {
		o.device = deviceFor(o.username)
	}

	servers, err := login.FetchServerList(ctx, login.Config{
		Address:  net.JoinHostPort(o.host, strconv.Itoa(o.port)),
		Username: o.username,
		Password: o.password,
		Pattern:  o.pattern,
		Device:   login.DefaultDevice(o.device),
		Timeout:  o.timeout,
	})
	if err != nil {
		return err
	}

	if o.jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(servers)
	}
	return printTable(servers)
}

func deviceFor(username string) string {
	sum := sha256.Sum256([]byte(deviceSalt + strings.ToLower(username)))
	return hex.EncodeToString(sum[:16])
}

func printTable(servers []serverlist.Server) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tGRUP\tSUNUCU\tADRES\tPORT")
	for _, s := range servers {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\n", s.ID, s.Group, s.Name, s.Host, s.Port)
	}
	return w.Flush()
}
