package dg1670aexporter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/northbright/ctx/ctxcopy"

	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/net/html"
)

type client struct {
	client *http.Client
	url    string
}

func (c *client) fetch(ctx context.Context) (*modemData, error) {
	req, err := http.NewRequest(http.MethodGet, c.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := ctxhttp.Do(ctx, c.client, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("modem returned non-200")
	}
	buf := make([]byte, 2*1024*1024)
	var out bytes.Buffer
	if err := ctxcopy.Copy(ctx, &out, resp.Body, buf); err != nil {
		return nil, err
	}
	return parseResp(&out)
}

func parseResp(data io.Reader) (*modemData, error) {
	node, err := html.Parse(data)
	if err != nil {
		return nil, err
	}
	doc := goquery.NewDocumentFromNode(node)
	out := &modemData{
		ds: parseDownstream(doc),
		us: parseUpstream(doc),
	}
	return out, nil
}

func parseDownstream(doc *goquery.Document) []downstreamData {
	downstream := doc.Find("h4:contains('Downstream')").Next()
	var out []downstreamData
	_ = downstream.Find("tr").Each(func(i int, selection *goquery.Selection) {
		if i == 0 {
			return
		}
		data := selection.Find("td")
		ch := downstreamData{
			DCID:           mustNodeAsInt(data.Nodes[1]),
			Freq:           mustNodeAsFloat(data.Nodes[2]) * 1000,
			Power:          mustNodeAsFloat(data.Nodes[3]),
			SNR:            mustNodeAsFloat(data.Nodes[4]),
			Modulation:     mustNodeAsInt(data.Nodes[5]),
			Octets:         mustNodeAsInt(data.Nodes[6]),
			Correcteds:     mustNodeAsInt(data.Nodes[7]),
			Uncorrectables: mustNodeAsInt(data.Nodes[8]),
		}
		out = append(out, ch)
	})
	return out
}

func parseUpstream(doc *goquery.Document) []upstreamData {
	upstream := doc.Find("h4:contains('Upstream')").Next()
	var out []upstreamData
	_ = upstream.Find("tr").Each(func(i int, selection *goquery.Selection) {
		if i < 2 {
			return
		}
		data := selection.Find("td")
		us := upstreamData{
			UCID:        mustNodeAsInt(data.Nodes[1]),
			Freq:        mustNodeAsFloat(data.Nodes[2]),
			Power:       mustNodeAsFloat(data.Nodes[3]),
			ChannelType: data.Nodes[4].FirstChild.Data,
			SymbolRate:  mustNodeAsInt(data.Nodes[5]),
			Modulation:  mustNodeAsInt(data.Nodes[6]),
		}
		out = append(out, us)
	})
	return out
}

func mustNodeAsFloat(d *html.Node) float64 {
	data := strings.Split(d.FirstChild.Data, " ")[0]
	floatData, err := strconv.ParseFloat(data, 64)
	if err != nil {
		panic("bad HTML node from modem")
	}
	return floatData
}
func mustNodeAsInt(d *html.Node) int64 {
	data := strings.Replace(d.FirstChild.Data, "QAM", "", 1)
	data = strings.Split(data, " ")[0]
	intData, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		panic("bad HTML node from modem")
	}
	return intData
}

type modemData struct {
	ds []downstreamData
	us []upstreamData
}

type downstreamData struct {
	DCID           int64
	Freq           float64
	Power          float64
	SNR            float64
	Modulation     int64
	Octets         int64
	Correcteds     int64
	Uncorrectables int64
}

type upstreamData struct {
	UCID        int64
	Freq        float64
	Power       float64
	ChannelType string
	SymbolRate  int64
	Modulation  int64
}
