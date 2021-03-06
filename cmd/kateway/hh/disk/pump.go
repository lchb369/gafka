package disk

import (
	"time"

	"github.com/funkygao/gafka/cmd/kateway/store"
	log "github.com/funkygao/log4go"
)

func (q *queue) pump() {
	defer func() {
		q.cursor.dump()
		q.wg.Done()
	}()

	log.Trace("queue[%s] start pump...", q.ident())

	var (
		b          block
		err        error
		partition  int32
		offset     int64
		okN, failN int64
		retries    int
		backoff    time.Duration
	)
	for {
		select {
		case <-q.quit:
			log.Trace("queue[%s] pump done, delivered: %d/%d", q.ident(), okN, failN)
			return
		default:
		}

		backoff = initialBackoff

		err = q.Next(&b)
		switch err {
		case nil:
			for retries = 0; retries < defaultMaxRetries; retries++ {
				// TODO we might use AsyncPub
				partition, offset, err = store.DefaultPubStore.SyncPub(q.clusterTopic.cluster, q.clusterTopic.topic, b.key, b.value)
				if err == nil {
					if Auditor != nil {
						Auditor.Trace("queue[%s] {P:%d O:%d}", q.ident(), partition, offset)
					}

					q.cursor.commitPosition()
					okN++
					q.inflights.Add(-1)
					q.deliverN.Add(1)
					if okN%dumpPerBlocks == 0 {
						if e := q.cursor.dump(); e != nil {
							log.Error("queue[%s] dump: %s", q.ident(), e)
						}
					}
					break
				} else if err == store.ErrInvalidTopic || err == store.ErrInvalidCluster {
					q.cursor.commitPosition()
					failN++
					q.deliverN.Add(1)
					q.inflights.Add(-1)
					err = nil // move ahead without retry
					break
				}

				log.Debug("queue[%s] {k:%s v:%s} %s", q.ident(), string(b.key), string(b.value), err)

				// backoff
				select {
				case <-q.quit:
					log.Trace("queue[%s] pump done, delivered: %d/%d", q.ident(), okN, failN)
					return
				case <-timer.After(backoff):
				}

				backoff *= 2
				if backoff >= maxBackoff {
					backoff = maxBackoff
				}
			}

			if err == nil {
				continue
			}

			// failed to deliver
			if err = q.Rollback(&b); err != nil {
				// should never happen
				log.Warn("queue[%s] skipped block <%s/%s>", q.ident(), string(b.key), string(b.value))

				failN++
			}

		case ErrQueueNotOpen:
			return

		case ErrCursorOutOfRange:
			log.Error(err.Error()) // TODO

		case ErrEOQ:
			select {
			case <-q.quit:
				log.Trace("queue[%s] pump done, delivered: %d/%d", q.ident(), okN, failN)
				return
			case <-timer.After(pollSleep):
			}

		default:
			log.Error("queue[%s] pump: %s +%v", q.ident(), err, q.cursor.pos)
		}
	}
}
