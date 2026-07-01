package data

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/config"
)

func TestPhoenixAlarmEventPayloadMatchesCapture(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")
	provider.controlSource = "W11WS2V3"
	provider.controlLocalIP = "10.32.1.48"
	provider.controlRemote = &net.UDPAddr{IP: net.ParseIP("10.32.1.200"), Port: 5057}

	got := string(provider.phoenixAlarmEventPayload(
		"L00029",
		749731,
		2,
		"1\n",
		"Підлипний А.М",
		time.Date(2026, 5, 16, 22, 27, 19, 0, time.Local),
	))
	want := "0[*]W11WS2V3[*]OP[*]10.32.1.200[*]CU[*]EVENT[*]L00029[*]1[*]0[*][*][*]749731[*]2[*]16.05.2026 22:27:19[*]1\n[*]10.32.1.200[*]0[*]10.32.1.48[*]Підлипний А.М[*]"
	if got != want {
		t.Fatalf("payload = %q, want %q", got, want)
	}
}

func TestPhoenixLoginAndPresencePayloadsMatchCapture(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")
	provider.controlSource = "W11WS2V3"
	provider.controlLocalIP = "10.32.1.48"
	provider.controlRemote = &net.UDPAddr{IP: net.ParseIP("10.32.1.200"), Port: 5057}
	now := time.Date(2026, 5, 19, 14, 54, 52, 0, time.Local)

	login := string(provider.phoenixLoginPayload(1998, "Підлипний А.М", now))
	wantLogin := "0[*]W11WS2V3[*]OP[*]10.32.1.200[*]CU[*]USERLOGIN[*][*]0[*]0[*][*][*]0[*]0[*]19.05.2026 14:54:52[*]Підлипний А.М\r\n1998\r\n10.32.1.48\r\n[*]10.32.1.200[*]0[*]10.32.1.48[*][*]"
	if login != wantLogin {
		t.Fatalf("USERLOGIN = %q, want %q", login, wantLogin)
	}

	ping := string(provider.phoenixPresencePayload("PING", "Підлипний А.М", now))
	if !strings.Contains(ping, "[*]PING[*]") || !strings.Contains(ping, phoenixPingMarker) {
		t.Fatalf("PING payload = %q", ping)
	}
}

func TestPhoenixControlAckMirrorsNotification(t *testing.T) {
	packet := parsePhoenixControlPacket([]byte(
		"0[*]PHOENIXSRV01[*]CU[*][*]OP[*]NEWEVENT[*]L00032[*]0",
	))
	got := string(phoenixControlAck(packet, &net.UDPAddr{
		IP:   net.ParseIP("10.32.1.200"),
		Port: 5057,
	}))
	want := "1[*]PHOENIXSRV01[*]CU[*]10.32.1.200[*]OP[*]NEWEVENT[*]L00032[*]0"
	if got != want {
		t.Fatalf("ACK = %q, want %q", got, want)
	}
}

func TestPhoenixControlSessionUsesDutyPortAndWaitsForAck(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	clientPort := availableUDPPort(t)

	provider := NewPhoenixDataProvider(nil, "")
	provider.ConfigureAlarmOperator(1998, "Оператор", "127.0.0.1", PhoenixRuntimeMetadata{
		ControlPort: server.LocalAddr().(*net.UDPAddr).Port,
		ClientPort:  clientPort,
	}, config.PhoenixClientRoleDuty)
	if err := provider.startControlCenterSession(); err != nil {
		t.Fatal(err)
	}
	defer provider.Shutdown()

	serverResult := make(chan error, 1)
	go func() {
		buf := make([]byte, 4096)
		_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, remote, err := server.ReadFromUDP(buf)
		if err != nil {
			serverResult <- err
			return
		}
		if remote.Port != clientPort {
			serverResult <- &unexpectedUDPPortError{got: remote.Port, want: clientPort}
			return
		}
		packet := parsePhoenixControlPacket(buf[:n])
		if _, err = server.WriteToUDP(phoenixControlAck(packet, server.LocalAddr().(*net.UDPAddr)), remote); err != nil {
			serverResult <- err
			return
		}
		notification := []byte("0[*]PHOENIXSRV01[*]CU[*][*]OP[*]NEWEVENT[*]L00029[*]0")
		if _, err = server.WriteToUDP(notification, remote); err != nil {
			serverResult <- err
			return
		}
		n, _, err = server.ReadFromUDP(buf)
		if err == nil && !parsePhoenixControlPacket(buf[:n]).ack {
			err = &missingPhoenixACKError{}
		}
		serverResult <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = provider.sendAlarmState(ctx, "L00029", 749731, 1, "", "Оператор", false)
	if err != nil {
		t.Fatalf("sendAlarmState() error = %v", err)
	}
	if err := <-serverResult; err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for provider.controlRevision.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if provider.controlRevision.Load() < 2 {
		t.Fatalf("control revision = %d, want ACK and notification updates", provider.controlRevision.Load())
	}
}

func availableUDPPort(t *testing.T) int {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

type unexpectedUDPPortError struct {
	got  int
	want int
}

type missingPhoenixACKError struct{}

func (*missingPhoenixACKError) Error() string {
	return "Phoenix notification response is not an ACK"
}

func (e *unexpectedUDPPortError) Error() string {
	return "UDP source port = " + strconv.Itoa(e.got) + ", want " + strconv.Itoa(e.want)
}
