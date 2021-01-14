package everquest

// EverQuest client interaction

import (
	"context"
	"errors"
	"fmt"
	"github.com/GontikR99/buffbot/internal/storage"
	"github.com/lxn/walk"
	"log"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	escCount     = 5 // Number of times to press ESC to clear
	newLineDelay = 30 * time.Millisecond
)

type Client struct {
	Config  storage.ControllerConfig
	Context context.Context

	logChan    <-chan EqLogEntry
	logSync    sync.Mutex
	nextLogTap int
	logTaps    map[int]*tapPair

	typeSync      sync.Mutex
	turnDirection int

	movingForward bool
}

// Create a client to interact with EverQuest
func NewEqClient(ctx context.Context, config storage.ControllerConfig) (client *Client, err error) {
	client = &Client{}
	client.Config = config
	client.Context = ctx
	client.logChan, err = readAllLogs(ctx, client.Config.EverQuestDirectory()+"/Logs")
	if err != nil {
		return
	}

	client.logTaps = make(map[int]*tapPair)
	go client.forwardLogMessages()
	return
}

// Receive messages from the log, send them on to all taps.  Maintain taps, too.
func (eqc *Client) forwardLogMessages() {
	for {
		select {
		case msg := <-eqc.logChan:
			eqc.logSync.Lock()
			for k, tap := range eqc.logTaps {
				select {
				case <-tap.done:
					delete(eqc.logTaps, k)
				default:
					select {
					case tap.messages <- msg:
					default:
						log.Printf("Tap %d is full", k)
					}
				}
			}
			eqc.logSync.Unlock()
		case <-eqc.Context.Done():
			return
		}
	}
}

type tapPair struct {
	messages chan<- EqLogEntry
	done     <-chan struct{}
}

// Allocate a new tap, receiving all new messages from the client from this point on.  When messages are
// no longer required, send something to `done`
func (eqc *Client) TapLog() (messages <-chan EqLogEntry, done func()) {
	mc := make(chan EqLogEntry, 4096)
	dc := make(chan struct{}, 2)
	entry := tapPair{mc, dc}
	eqc.logSync.Lock()
	eqc.logTaps[eqc.nextLogTap] = &entry
	eqc.nextLogTap += 1
	eqc.logSync.Unlock()
	return mc, func() { dc <- struct{}{} }
}

var (
	nameRE = regexp.MustCompile("^[a-zA-Z]+$")
)

func (eqc *Client) Tell(who string, what string) error {
	if !nameRE.MatchString(who) {
		return errors.New("Not a valid name in Tell")
	}
	if strings.IndexByte(what, '\n') != -1 || strings.IndexByte(what, 0x1b) != -1 {
		return errors.New("Not a valid message in Tell")
	}
	return eqc.Send("/tell " + who + " " + what)
}

func (eqc *Client) Tellf(who string, fmtstr string, args ...interface{}) error {
	return eqc.Tell(who, fmt.Sprintf(fmtstr, args...))
}

type EqInput struct {
	client *Client
}

func (eqc *Client) GrabInput() (result EqInput, err error) {
	eqc.typeSync.Lock()
	err = raiseEverquest()
	if err != nil {
		time.Sleep(100 * time.Millisecond)
		eqc.typeSync.Unlock()
		return
	}
	result = EqInput{eqc}
	return
}

func (eqc *Client) Raise() error {
	return raiseEverquest()
}

// Send some text to EverQuest
func (eqc *Client) Send(parts ...interface{}) error {
	eqi, err := eqc.GrabInput()
	if err != nil {
		return err
	}
	defer eqi.Release()
	return eqi.Send(parts...)
}

// Ensure that we don't have a chat going on.
func (eqc *Client) CloseChat() error {
	eqi, err := eqc.GrabInput()
	if err != nil {
		return err
	}
	defer eqi.Release()
	return eqi.CloseChat()
}

func (eqc *Client) Tap(key rune) error {
	eqi, err := eqc.GrabInput()
	if err != nil {
		return err
	}
	defer eqi.Release()
	return eqi.Tap(key)
}

func (eqi EqInput) Release() {
	eqi.client.typeSync.Unlock()
	eqi.client = nil
}

// Start turning left or right
func (eqi EqInput) StartTurn(direction int) error {
	if eqi.client.turnDirection < 0 && direction >= 0 {
		submitKbMouse(func() {
			pressKey(vkD)
			time.Sleep(tapDelay)
			releaseKey(vkD)
		})
	} else if eqi.client.turnDirection > 0 && direction <= 0 {
		submitKbMouse(func() {
			pressKey(vkA)
			time.Sleep(tapDelay)
			releaseKey(vkA)
		})
	}
	eqi.client.turnDirection = direction
	if eqi.client.turnDirection < 0 {
		submitKbMouse(func() {
			pressKey(vkD)
		})
	} else if eqi.client.turnDirection > 0 {
		submitKbMouse(func() {
			pressKey(vkA)
		})
	}
	return nil
}

// End the current turn
func (eqi EqInput) EndTurn() error {
	if eqi.client.turnDirection == 0 {
		return nil
	}
	if eqi.client.turnDirection < 0 {
		submitKbMouse(func() {
			pressKey(vkD)
			time.Sleep(tapDelay)
			releaseKey(vkD)
		})
	} else if eqi.client.turnDirection > 0 {
		submitKbMouse(func() {
			pressKey(vkA)
			time.Sleep(tapDelay)
			releaseKey(vkA)
		})
	}
	eqi.client.turnDirection = 0
	return nil
}

// Start moving forward
func (eqi EqInput) StartForward() error {
	eqi.client.movingForward = true
	submitKbMouse(func() {
		pressKey(vkW)
	})
	return nil
}

// Stop moving forward
func (eqi EqInput) EndForward() error {
	if eqi.client.movingForward {
		submitKbMouse(func() {
			pressKey(vkW)
			time.Sleep(tapDelay)
			releaseKey(vkW)
		})
	}
	eqi.client.movingForward = false
	return nil
}

// Press the ESC key a bunch of times to close down any temporary windows
func (eqi EqInput) ClearWindows() {
	submitKbMouse(func() {
		for i := 0; i < escCount; i++ {
			tap('\x1b')
		}
	})
}

func (eqc *Client) ClearWindows() error {
	eqi, err := eqc.GrabInput()
	if err != nil {
		log.Println(err)
		return err
	}
	defer eqi.Release()
	eqi.ClearWindows()
	return nil
}

// Make sure we're not currently typing into a chat window
func (eqi EqInput) CloseChat() error {
	select {
	case <-eqi.client.Context.Done():
		return errors.New("Client is shut down")
	default:
		break
	}
	submitKbMouse(func() {
		// Ensure we've got a totally clear entry line.
		tapSlow('\n')
		time.Sleep(newLineDelay)
		tapSlow('\n')
		time.Sleep(newLineDelay)
		tapSlow('/')
		time.Sleep(newLineDelay)
		tapSlow('\b')
		time.Sleep(newLineDelay)
		tapSlow('\n')
		time.Sleep(newLineDelay)
	})
	return nil
}

func (eqi EqInput) Tap(key rune) error {
	select {
	case <-eqi.client.Context.Done():
		return errors.New("Client is shut down")
	default:
		break
	}
	submitKbMouse(func() {
		// Ensure we've got a totally clear entry line.
		tapSlow(key)
	})
	return nil
}

// Send something to the interface.  Can either be strings, or callbacks.
func (eqi EqInput) Send(parts ...interface{}) error {
	select {
	case <-eqi.client.Context.Done():
		return errors.New("Client is shut down")
	default:
		break
	}
	submitKbMouse(func() {
		// Ensure we've got a totally clear entry line.
		tapSlow('\n')
		time.Sleep(newLineDelay)
		tapSlow('\n')
		time.Sleep(newLineDelay)
		tapSlow('/')
		time.Sleep(newLineDelay)
		tapSlow('\b')
		time.Sleep(newLineDelay)
	})
	for _, part := range parts {
		switch v := part.(type) {
		case string:
			submitKbMouse(func() {
				walk.Clipboard().Clear()
				walk.Clipboard().SetText(v)
				tapCtrl('v')
				//typewrite(v)
			})
		case func(EqInput):
			v(eqi)
		default:
			return fmt.Errorf("Don't know how to deal with a %v", v)
		}
	}
	submitKbMouse(func() {
		tap('\n')
	})
	return nil
}

// Click at the specified point in the client
func (eqi EqInput) ClickAt(x int, y int) (err error) {
	select {
	case <-eqi.client.Context.Done():
		return errors.New("Client is shut down")
	default:
		break
	}
	submitKbMouse(func() {
		l, t, _, _, err := getEqClientArea()
		if err != nil {
			return
		}
		err = moveMouse(x+l, y+t)
		if err != nil {
			return
		}
		err = leftClick()
	})
	return
}

func (eqc *Client) ClickAt(x int, y int) error {
	eqi, err := eqc.GrabInput()
	if err != nil {
		return err
	}
	defer eqi.Release()
	return eqi.ClickAt(x, y)
}

// Reserve an OS thread to sending keyboard/mouse events, so that its scheduling isn't
// subject to being preempted by other go-ings on.
type inputJob struct {
	callback func()
	done     chan struct{}
}

var jobChan = make(chan inputJob)

func init() {
	runtime.LockOSThread()
	go func() {
		runtime.LockOSThread()
		for {
			ij := <-jobChan
			ij.callback()
			ij.done <- struct{}{}
		}
	}()
}

func submitKbMouse(callback func()) {
	ij := inputJob{callback, make(chan struct{})}
	jobChan <- ij
	<-ij.done
}
