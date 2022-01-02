package logic

import (
	"fmt"
	"mydns/master/internal/svc"
	"mydns/master/internal/types"
	"os"
	"path"
	"sort"
	"text/template"
)

type zone struct {
	Domain     string
	Serial     int64
	MainNS     string
	Nameserver []string
	Records    []types.Record
}

func genZoneFile(z zone, sc *svc.ServiceContext) error {
	for _, i := range z.Records {
		if i.RecordType == "NS" {
			z.Nameserver = append(z.Nameserver, i.PointsTo)
		}
	}
	if len(z.Nameserver) == 0 {
		return fmt.Errorf("NS type not found in records")
	}

	fmt.Printf("nameserver: %#v\tlength: %d", z.Nameserver, len(z.Nameserver))

	// sort to find main NS
	sort.Slice(z.Nameserver, func(i, j int) bool {
		return z.Nameserver[i] < z.Nameserver[j]
	})
	fmt.Printf("nameserver: %#v\tlength: %d", z.Nameserver, len(z.Nameserver))
	z.MainNS = z.Nameserver[0]
	fmt.Printf("main NS: %s\n", z.MainNS)

	tplParse, err := template.ParseFiles(sc.Config.ZoneTemplate)
	if err != nil {
		return err
	}

	zf, err := os.Create(path.Join(sc.Config.ZoneSavePath, z.Domain+".zone"))
	if err != nil {
		return err
	}
	defer zf.Close()

	err = tplParse.Execute(zf, z)
	if err != nil {
		return err
	}

	return nil
}
