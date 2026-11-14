package workbench

import (
	"sync"

	"github.com/vivaui/viva/app"
)

// transferSleepReason is what the system's power diagnostics show while a
// transfer holds the machine awake.
const transferSleepReason = "Motty: file transfer"

// transferSleep is how many windows of this process have a transfer moving
// bytes. One system request serves them all, and it exists only while the count
// is above zero, so no timer or goroutine runs when nothing is transferring.
var transferSleep struct {
	mu      sync.Mutex
	windows int
	release func()
}

// holdTransferSleep adds or removes one window's claim. A system that refuses
// the request leaves transfers working as before; staying awake is an
// optimization, not a requirement of copying files.
func holdTransferSleep(held bool) {
	transferSleep.mu.Lock()
	defer transferSleep.mu.Unlock()
	if held {
		transferSleep.windows++
	} else if transferSleep.windows > 0 {
		transferSleep.windows--
	}
	switch {
	case transferSleep.windows > 0 && transferSleep.release == nil:
		if release, err := app.PreventSleep(transferSleepReason); err == nil {
			transferSleep.release = release
		}
	case transferSleep.windows == 0 && transferSleep.release != nil:
		transferSleep.release()
		transferSleep.release = nil
	}
}

// keepAwakeForTransfers follows this window's running transfers: the first adds
// its claim and the last one takes it away, so a paused or finished transfer
// does not keep a laptop awake.
func (state *State) keepAwakeForTransfers(active bool) {
	center := &state.transfers
	if center.awake == active {
		return
	}
	center.awake = active
	holdTransferSleep(active)
}
