// +build windows

package main

import (
	"encoding/json"
	"fmt"
	"github.com/kardianos/service"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type Config struct {
	Name, DisplayName, Description string

	OECPath string
	Args    []string
	Env     []string

	Stderr, Stdout string
}

var logger service.Logger

type Program struct {
	*Config

	service service.Service
	cmd     *exec.Cmd
}

func (p *Program) Start(s service.Service) error {
	p.cmd = exec.Command("cmd", append([]string{"/C", p.OECPath}, p.Args...)...)
	p.cmd.Env = append(os.Environ(), p.Env...)

	go p.run()
	return nil
}
func (p *Program) run() {
	logger.Info("Starting ", p.DisplayName)

	if p.Stderr != "" {
		stderr, err := os.OpenFile(p.Stderr, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
		if err != nil {
			logger.Errorf("Failed to open std err %q: %v", p.Stderr, err)
			p.service.Stop()
			return
		}
		defer stderr.Close()
		p.cmd.Stderr = stderr
	}
	if p.Stdout != "" {
		stdout, err := os.OpenFile(p.Stdout, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
		if err != nil {
			logger.Errorf("Failed to open std out %q: %v", p.Stdout, err)
			p.service.Stop()
			return
		}
		defer stdout.Close()
		p.cmd.Stdout = stdout
	}

	err := p.cmd.Run()
	if err != nil {
		logger.Errorf("Failed to run OEC: %v", err)
	}

	p.service.Stop()
	return
}

func errMessageCtrlC(errType string, err error) []byte {
	return []byte(fmt.Sprintf("Failed while stopping the service. %s error: %v\n", errType, err))
}

func sendCtrlCEvent(pid int, stderr string) {
	file, err := os.OpenFile(stderr, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		logger.Warningf("Failed to open std out %q: %v", stderr, err)
	}
	defer file.Close()

	kernel32, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		file.Write(errMessageCtrlC("LoadDLL", err))
	}
	defer kernel32.Release()

	freeConsole, err := kernel32.FindProc("FreeConsole")
	if err != nil {
		file.Write(errMessageCtrlC("FindProc[FreeConsole]", err))
	}
	r, _, err := freeConsole.Call()
	if r == 0 {
		file.Write(errMessageCtrlC("FreeConsole", err))
	}

	attachConsole, err := kernel32.FindProc("AttachConsole")
	if err != nil {
		file.Write(errMessageCtrlC("FindProc[AttachConsole]", err))
	}
	r, _, err = attachConsole.Call(uintptr(pid))
	if r == 0 {
		file.Write(errMessageCtrlC("AttachConsole", err))
	}

	setConsoleCtrlHandler, err := kernel32.FindProc("SetConsoleCtrlHandler")
	if err != nil {
		file.Write(errMessageCtrlC("FindProc[SetConsoleCtrlHandler]", err))
	}
	r, _, err = setConsoleCtrlHandler.Call(0, 1)
	if r == 0 {
		file.Write(errMessageCtrlC("SetConsoleCtrlHandler", err))
	}

	generateConsoleCtrlEvent, err := kernel32.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		file.Write(errMessageCtrlC("FindProc[GenerateConsoleCtrlEvent]", err))
	}
	r, _, err = generateConsoleCtrlEvent.Call(syscall.CTRL_C_EVENT, uintptr(pid))
	if r == 0 {
		file.Write(errMessageCtrlC("GenerateConsoleCtrlEvent", err))
	}
}

func (p *Program) Stop(s service.Service) error {
	// already terminated ungracefully
	if p.cmd.ProcessState != nil {
		return nil
	}
	// terminate gracefully
	sendCtrlCEvent(p.cmd.Process.Pid, p.Stderr)
	return nil
}

func getConfigPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	ext := filepath.Ext(exePath)
	configPath := exePath[:len(exePath)-len(ext)] + ".json"

	return configPath, nil
}

func getConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	conf := &Config{}

	err = json.NewDecoder(file).Decode(&conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func validateConfig(config *Config) {
	if config.Stderr != "" {
		stderr, err := os.OpenFile(config.Stderr, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
		if err != nil {
			log.Fatalf("Failed to open std err %q: %v", config.Stderr, err)
		}
		defer stderr.Close()
	}
	if config.Stdout != "" {
		stdout, err := os.OpenFile(config.Stdout, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
		if err != nil {
			log.Fatalf("Failed to open std out %q: %v", config.Stdout, err)
		}
		defer stdout.Close()
	}

	_, err := os.Stat(config.OECPath)
	if err != nil {
		log.Fatalf("Failed to find OEC executable %q: %v", config.OECPath, err)
	}
}

func main() {

	configPath, err := getConfigPath()
	if err != nil {
		log.Fatal(err)
	}
	config, err := getConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	svcConfig := &service.Config{
		Name:        config.Name,
		DisplayName: config.DisplayName,
		Description: config.Description,
	}

	prg := &Program{
		Config: config,
	}

	svc, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	prg.service = svc

	errs := make(chan error, 5)
	logger, err = svc.Logger(errs)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			err := <-errs
			if err != nil {
				log.Print(err)
			}
		}
	}()

	if len(os.Args) > 1 {
		action := os.Args[1]

		if action == "start" {
			validateConfig(config)
		}
		err := service.Control(svc, action)
		if err != nil {
			if strings.Contains(err.Error(), "Unknown action") {
				log.Println(err)
				log.Fatalf("Valid actions: %q\n", service.ControlAction)
			}
			log.Fatal(err)
		}
		if action == "stop" {
			log.Println("OEC service stopped successfully.")
		} else {
			log.Println("OEC service " + action + "ed successfully.")
		}

		return
	}

	err = svc.Run()
	if err != nil {
		logger.Error(err)
	}
}