package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/emicklei/proto"
)

func getProtoReader(p string) io.ReadCloser {
	if strings.HasPrefix(p, "http") {
		return getFromHTTP(p)
	}
	f, err := os.Open(p)
	must(err)
	return f
}

func getFromHTTP(url string) io.ReadCloser {
	resp, err := http.Get(url)
	must(err)
	if resp.StatusCode != 200 {
		log.Fatalf("unexpected status %v", resp.StatusCode)
	}
	return resp.Body
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("Usage: genfetch <path|url>")
	}
	rc := getProtoReader(os.Args[1])
	defer rc.Close()
	parser := proto.NewParser(rc)
	defs, err := parser.Parse()
	must(err)
	rpcs := []method{}
	pkgName := ""
	svcName := ""
	proto.Walk(defs, func(v proto.Visitee) {
		pkg, ok := v.(*proto.Package)
		if ok {
			pkgName = pkg.Name
		}
		svc, ok := v.(*proto.Service)
		if ok {
			svcName = svc.Name
		}
	})

	proto.Walk(defs, proto.WithRPC(func(rpc *proto.RPC) {
		url := "/twirp/" + pkgName + "." + svcName + "/" + rpc.Name
		rpcs = append(rpcs, method{rpc.Name, url})
	}))

	f, err := os.Create("client.js")
	must(err)
	defer f.Close()

	t := template.Must(template.New("").Parse(tmpl))
	err = t.Execute(f, rpcs)
	must(err)
}

type method struct {
	Name, URL string
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

const tmpl = `export default { {{range . }}
	async {{ .Name }}(req) {
		const resp = await fetch('{{.URL}}', {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			credentials: 'include',
			body: JSON.stringify(req || {}),
		});
		const json = await resp.json();
		return json;
	},
	{{- end }}
}
`
