package manager

import (
	"context"
	"fmt"
	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/Filecoin-Titan/titan-container/db"
	"github.com/Filecoin-Titan/titan-container/node/modules/dtypes"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	maxTXTRecord        = 10000
	txtRecordExpireTime = 30 * time.Minute
)

type TXTRecord struct {
	time  time.Time
	value string
}

type DNSServer struct {
	DNSServerAddress dtypes.DNSServerAddress
	DB               *db.ManagerDB
	txtRecords       *SafeMap
}

func NewDNSServer(db *db.ManagerDB, address dtypes.DNSServerAddress) *DNSServer {
	dnsServer := &DNSServer{DB: db, DNSServerAddress: address, txtRecords: NewSafeMap()}
	go dnsServer.start()

	return dnsServer
}

func (ds *DNSServer) start() {
	handler := &dnsHandler{dnsServer: ds}
	server := &dns.Server{
		Addr:      string(ds.DNSServerAddress),
		Net:       "udp",
		Handler:   handler,
		UDPSize:   65535,
		ReusePort: true,
	}

	log.Debugf("Starting DNS server on %s", ds.DNSServerAddress)
	err := server.ListenAndServe()
	if err != nil {
		log.Errorf("Failed to start dns server: %s\n", err.Error())
	}
}

func (ds *DNSServer) deleteExpireTXTRecord() {
	toBeDeleteDomain := make([]string, 0)
	if ds.txtRecords.Len() >= maxTXTRecord {
		ds.txtRecords.Range(func(key, value interface{}) {
			// can not delete key in this range
			domain := key.(string)
			record := value.(TXTRecord)
			if record.time.Add(txtRecordExpireTime).Before(time.Now()) {
				toBeDeleteDomain = append(toBeDeleteDomain, domain)
			}
		})
	}

	if len(toBeDeleteDomain) > 0 {
		for _, domain := range toBeDeleteDomain {
			ds.txtRecords.Delete(domain)
		}
	}
}
func (ds *DNSServer) SetTXTRecord(domain string, value string) error {
	// release record
	ds.deleteExpireTXTRecord()

	if ds.txtRecords.Len() >= maxTXTRecord {
		return fmt.Errorf("current TXTRecord %d, out of max txt record", ds.txtRecords.Len())
	}

	domain = strings.TrimSuffix(domain, ".")
	ds.txtRecords.Set(domain, &TXTRecord{time: time.Now(), value: value})

	return nil
}

func (ds *DNSServer) DeleteTXTRecord(domain string) error {
	ds.txtRecords.Delete(domain)
	return nil
}

type dnsHandler struct {
	dnsServer *DNSServer
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	switch r.Opcode {
	case dns.OpcodeQuery:
		h.HandlerQuery(m, w.RemoteAddr().String())
	}

	if err := w.WriteMsg(m); err != nil {
		log.Errorf("dns server write message error %s", err.Error())
	}
}

func (h *dnsHandler) HandlerQuery(m *dns.Msg, remoteAddr string) {
	log.Debugf("HandlerQuery request %#v", *m)
	for _, q := range m.Question {
		domain := strings.TrimSuffix(strings.ToLower(q.Name), ".")
		switch q.Qtype {
		case dns.TypeTXT:
			if err := h.handlerTXTRecord(m, domain); err != nil {
				log.Errorf("handlerTXTRecord error %s", err.Error())
			}
		case dns.TypeCAA:
			if err := h.handlerCAARecord(m, domain); err != nil {
				log.Errorf("handlerCAARecord error %s", err.Error())
			}
		case dns.TypeA:
			log.Debugf("Query for %s, remote address %s\n", q.Name, remoteAddr)

			if ok, err := h.ReserveHostnames(m, domain); err != nil {
				log.Infof("ReserveHostnames %s", err.Error())
				return
			} else if ok {
				return
			}

		}
	}
}

func (h *dnsHandler) handlerTXTRecord(m *dns.Msg, domain string) error {
	value, ok := h.dnsServer.txtRecords.Get(domain)
	if ok {
		record := value.(*TXTRecord)

		txt := &dns.TXT{}
		txt.Hdr = dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60}
		txt.Txt = []string{record.value}

		m.Answer = append(m.Answer, txt)
		return nil
	}
	return fmt.Errorf("can not find %s txt record", domain)
}

func (h *dnsHandler) handlerCAARecord(m *dns.Msg, domain string) error {
	rr, err := dns.NewRR(fmt.Sprintf("%s CAA 0 issue letsencrypt.org", domain))
	if err == nil {
		m.Answer = append(m.Answer, rr)
	}
	return err
}

func (h *dnsHandler) ReserveHostnames(m *dns.Msg, domain string) (bool, error) {
	fields := strings.Split(domain, ".")
	if len(fields) < 4 {
		return false, fmt.Errorf("invalid domain %s", domain)
	}

	providerId := fields[0]

	provider, err := h.dnsServer.DB.GetProviderById(context.Background(), types.ProviderID(providerId))
	if err != nil {
		return false, err
	}

	if rr, err := dns.NewRR(fmt.Sprintf("%s A %s", domain, provider.IP)); err == nil {
		m.Answer = append(m.Answer, rr)
	}
	return true, nil
}
