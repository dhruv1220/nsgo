package restart

import (
	"os"
	"testing"
	"time"
)

func TestRestartGuardMarksRestartedProcess(t *testing.T) {
	was := restartedAt.Load()
	t.Cleanup(func() { restartedAt.Store(was) })
	restartedAt.Store(0)
	t.Setenv(guardEnv, "1")
	markRestarted()
	if !Looping() {
		t.Fatal("restart guard did not mark the restarted process")
	}
	if _, ok := os.LookupEnv(guardEnv); ok {
		t.Fatal("restart guard environment leaked into the app")
	}
}

func TestRestartGuardRefusesOnlyARestartAtLaunch(t *testing.T) {
	was := restartedAt.Load()
	t.Cleanup(func() { restartedAt.Store(was) })
	restartedAt.Store(time.Now().UnixNano())
	if !Looping() {
		t.Fatal("an app restarting as it launched was allowed to launch another process")
	}
	restartedAt.Store(time.Now().Add(-LoopWindow - time.Second).UnixNano())
	if Looping() {
		t.Fatal("a restarted app was refused a restart the user asked for later")
	}
	restartedAt.Store(0)
	if Looping() {
		t.Fatal("an ordinary release app was refused a restart")
	}
}
