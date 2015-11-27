package main

import (
	"net/http"
	"strconv"
	"time"

	log "github.com/funkygao/log4go"
	"github.com/gorilla/mux"
)

// /{ver}/topics/{topic}/{group}/{id}?offset=n&limit=1&timeout=10m
// TODO offset manager, flusher, partitions, join group
func (this *Gateway) subHandler(w http.ResponseWriter, req *http.Request) {
	if !this.authenticate(req) {
		this.writeAuthFailure(w)
		return
	}

	if this.breaker.Open() {
		this.writeBreakerOpen(w)
		return
	}

	var (
		ver     string
		topic   string
		group   string
		timeout time.Duration = time.Duration(time.Hour)
		err     error
		limit   int = 1
	)

	limitParam := req.URL.Query().Get("limit")
	timeoutParam := req.URL.Query().Get("timeout")
	if limitParam != "" {
		limit, err = strconv.Atoi(limitParam)
		if err != nil {
			this.writeBadRequest(w)

			log.Error("%s Sub {topic:%s, group:%s, limit:%s, timeout:%s} %v",
				req.RemoteAddr, topic, group, limitParam, timeoutParam, err)
			return
		}
	}
	if timeoutParam != "" {
		timeout, err = time.ParseDuration(timeoutParam)
		if err != nil {
			this.writeBadRequest(w)

			log.Error("%s Sub {topic:%s, group:%s, limit:%s, timeout:%s} %v",
				req.RemoteAddr, topic, group, limitParam, timeoutParam, err)
			return
		}

		if timeout.Nanoseconds() == 0 {
			timeout = time.Duration(time.Hour * 24 * 3650) // TODO 10 years is enough?
		}
	}

	params := mux.Vars(req)
	ver = params["ver"]
	topic = params["topic"]
	group = params["group"]
	log.Info("%s Sub {topic:%s, group:%s, limit:%s, timeout:%s}",
		req.RemoteAddr, topic, group, limitParam, timeoutParam)

	if err = this.consume(ver, topic, limit, group, timeout, w, req); err != nil {
		this.breaker.Fail()
		log.Error("%s Sub {topic:%s, group:%s, limit:%s, timeout:%s} %v",
			req.RemoteAddr, topic, group, limitParam, timeoutParam, err)

		w.WriteHeader(http.StatusInternalServerError) // TODO
		w.Write([]byte(err.Error()))
	}
}

func (this *Gateway) consume(ver, topic string, limit int, group string,
	timeout time.Duration,
	w http.ResponseWriter, req *http.Request) error {
	cg, err := this.subPool.PickConsumerGroup(topic, group, req.RemoteAddr)
	if err != nil {
		return err
	}

	n := 0
	for {
		select {
		case <-time.After(timeout):
			return nil

		case msg := <-cg.Messages():
			if _, err := w.Write(msg.Value); err != nil {
				log.Warn("Sub killing consumer {topic:%s group:%s client:%s}",
					topic, group, req.RemoteAddr)
				go this.subPool.KillClient(topic, group, req.RemoteAddr)
				return err
			}

			// client really got this msg, safe to commit
			cg.CommitUpto(msg)

			if limit > 0 {
				n++
				if n >= limit {
					return nil
				}
			}

		case err := <-cg.Errors():
			log.Error("%s {topic:%s, group:%s}: %+v", req.RemoteAddr, topic, group, err)
		}
	}

	return nil

}
