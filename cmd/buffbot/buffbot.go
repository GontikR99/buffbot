package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/GontikR99/buffbot/internal/everquest"
	"github.com/GontikR99/buffbot/internal/storage"
	"github.com/GontikR99/buffbot/internal/ui"
	"log"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var tellRE = regexp.MustCompile("^([A-Za-z]+) (?:tells|told) you, '(.*)?'$")
var sayRE = regexp.MustCompile("^([A-Za-z]+) says, '(.*)?'$")
var yousayRE = regexp.MustCompile("^You say, 'Hail, (.*)?'$")

type BuffEntry struct {
	Name  string
	Index int
}

var buffs []BuffEntry

func allBuffs() string {
	buffStrings := []string{}
	for _, entry := range buffs {
		buffStrings = append(buffStrings, fmt.Sprintf("[%s]", entry.Name))
	}
	return strings.Join(buffStrings, ", ")
}

func CastBuff(eqc *everquest.Client, selfName, who string, buff string) error {
	eqi, err := eqc.GrabInput()
	if err != nil {
		return err
	}
	defer eqi.Release()

	eqi.Send("/tell " + who + " One moment, please.")

	eqi.ClearWindows()
	eqi.Send("/target " + who)
	<-time.After(500 * time.Millisecond)

	curTarget := func() string {
		localCtx, doneFunc := context.WithDeadline(eqc.Context, time.Now().Add(10*time.Second))
		defer doneFunc()

		tap, done := eqc.TapLog()
		rndbits := make([]byte, 4)
		rand.Read(rndbits)
		rndStr := hex.EncodeToString(rndbits)
		eqi.Send(";tell " + selfName + " %t " + rndStr)
		defer done()
		for {
			select {
			case <-localCtx.Done():
				return ""
			case inMsg := <-tap:
				parts := tellRE.FindStringSubmatch(inMsg.Message)
				if parts != nil && strings.Contains(parts[2], rndStr) {
					return strings.Split(parts[2], " ")[0]
				}
			}
		}
	}()

	if curTarget == "" || !strings.EqualFold(curTarget, who) {
		eqi.Send("/tell "+who+" Hrmm, I can't seem to target you.  Maybe come closer to me?  I'm giving up for now.")
		return nil
	} else {
		log.Println("My target is " + curTarget)
	}

	buff = strings.ToLower(buff)
	didBuff := false
	for _, entry := range buffs {
		if strings.Contains(buff, strings.ToLower(entry.Name)) {
			eqi.Send("/ttell Casting " + entry.Name)
			for j := 0; j < 5; j++ {
				eqi.Send("/cast " + strconv.Itoa(entry.Index))
				<-time.After(100 * time.Millisecond)
			}
			<-time.After(10 * time.Second)
			didBuff = true
		}
	}

	if !didBuff {
		eqi.Send("/ttell I don't recognize your request.  I can cast " + allBuffs())
	}
	return nil
}

func main() {
	runtime.GOMAXPROCS(16)
	cfg := &storage.BoltholdBackedConfig{}
	const advertKey = "advert"
	if !cfg.HasConfItem(advertKey) {
		cfg.SetConfItem(advertKey, "I'm a buff bot.  Send me a /TELL if you want any of:")
	}
	ui.RunMainWindow(cfg, []ui.ConfigurationLine{
		{"spell1", "Spell gem#1 keyword (leave empty if no buff loaded in this spell gem)"},
		{"spell2", "Spell gem#2 keyword"},
		{"spell3", "Spell gem#3 keyword"},
		{"spell4", "Spell gem#4 keyword"},
		{"spell5", "Spell gem#5 keyword"},
		{"spell6", "Spell gem#6 keyword"},
		{"spell7", "Spell gem#7 keyword"},
		{"spell8", "Spell gem#8 keyword"},
		{"spell9", "Spell gem#9 keyword"},
		{advertKey, "Text to /ooc periodically, or blank for no advertisement"},
	}, func(ctx context.Context, config storage.ControllerConfig) {
		buffs = []BuffEntry{}
		for i := 1; i <= 9; i++ {
			be := cfg.GetConfItem("spell" + strconv.Itoa(i))
			be = strings.TrimSpace(be)
			if be != "" {
				buffs = append(buffs, BuffEntry{
					Name:  be,
					Index: i,
				})
			}
		}

		eqc, err := everquest.NewEqClient(ctx, config)
		if err != nil {
			log.Println("Failed to initialize EverQuest client interface")
			return
		}

		eqc.ClearWindows()
		err = eqc.Send("/target group1")
		if err!=nil {
			log.Println(err)
		}
		<-time.After(200 * time.Millisecond)

		selfName := func() string {
			tap, done := eqc.TapLog()
			defer done()

			eqc.Tap('h')
			for {
				select {
				case <-eqc.Context.Done():
					return ""
				case inMsg := <-tap:
					parts := yousayRE.FindStringSubmatch(inMsg.Message)
					if parts != nil {
						return parts[1]
					}
				}
			}
		}()
		if selfName == "" {
			return
		}
		log.Println("My name is " + selfName)

		go func() {
			tap, done := eqc.TapLog()
			defer done()
			for {
				select {
				case msg := <-tap:
					parts := tellRE.FindStringSubmatch(msg.Message)
					if parts != nil && !strings.EqualFold(selfName, parts[1]) && !strings.Contains(strings.ToLower(parts[2]), "afk") && !strings.Contains(strings.ToLower(parts[2]), "away from") {
						CastBuff(eqc, selfName, parts[1], parts[2])
					}
				case <-eqc.Context.Done():
					return
				}
			}
		}()

		go func() {
			tap, done := eqc.TapLog()
			defer done()
			for {
				select {
				case msg := <-tap:
					parts := sayRE.FindStringSubmatch(msg.Message)
					if parts != nil && strings.EqualFold(parts[2], "Hail, "+selfName) {
						eqc.Send("/say I'm a buff bot.  Send me a /TELL if you want any of: " + allBuffs())
					}
				case <-eqc.Context.Done():
					return
				}
			}
		}()

		zoneMsgTap, zmdoneFunc := eqc.TapLog()
		defer zmdoneFunc()
		announce := time.After(10*time.Minute)
		for {
			select {
			case <-eqc.Context.Done():
				return
			case msg := <-zoneMsgTap:
				if strings.HasPrefix(msg.Message, "LOADING") {
					return
				}
			case <-announce:
				if cfg.GetConfItem(advertKey) != "" {
					eqc.Send("/ooc " + cfg.GetConfItem(advertKey) + " " + allBuffs())
				}
				announce = time.After(10 * time.Minute)
			}
		}
	})
}
