HNSSRC = hns/hns.go hns/ghns/ghns.go \
      hns/collector/collector.go hns/render/render.go \
      hns/config/config.go


all:	index.fcgi index.cgi debug server

clean:
	-rm -f index.fcgi index.cgi server cuihns

debug:	cuihns

server: cmd/server/main.go $(HNSSRC)
	go build -gcflags "-N -l" -o server ./cmd/server

index.fcgi:	cmd/fcgihns/main.go $(HNSSRC)
	go build -o index.fcgi ./cmd/fcgihns

index.cgi:	cmd/cgihns/main.go $(HNSSRC)
	go build -o index.cgi ./cmd/cgihns

cuihns:	cmd/cuihns/main.go $(HNSSRC)
	go build -gcflags "-N -l" -o cuihns ./cmd/cuihns
