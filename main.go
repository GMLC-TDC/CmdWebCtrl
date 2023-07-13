package main

import (
	"bufio"
	"fmt"
	"github.com/olahol/melody"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

var cmd *exec.Cmd

var stdout *bufio.Scanner
var stderr *bufio.Scanner
var logtext = ""

var melodyWs *melody.Melody

func mainPageResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")

	cmdStr := viper.GetString("Command")
	argArray := viper.GetStringSlice("Args")
	logCmdStr := getLogCmdStr(cmdStr, argArray)

	body := `
	<html>
	<head>
	<style>
		div#log_text {
			height: 500px;
			resize: vertical;
			overflow-y: auto;
			padding: 20px;
			border-style: inset;
			border-width: medium;
		}
	</style>
	<script type="text/javascript">
		window.onload = setupOnPageLoad;
		function setupOnPageLoad() {

			// Make the restart button functional
			const restartBtn = document.getElementById('restart-btn');
			restartBtn.addEventListener('click', async _ => {
				doRestart();
			});

			// Start websocket for getting log updates
			var url = "ws://" + window.location.host + "/logws";
			var ws = new WebSocket(url);
			var log_text = document.getElementById("log_text");
			ws.onmessage = function(msg) {
				doScroll = isAtBottom(log_text);

				console.log(msg.data);
				log_text.innerHTML += msg.data;

				if(doScroll) {
					scrollBottom(log_text);
				}
			};

			async function doRestart() {
				try {
					var data = new FormData(document.getElementById("restartForm"));
					console.log(data);
					const response = await fetch('/restart', {
						method: 'POST',
						body: data
					});
					console.log('Restarted', response);
				} catch(err) {
					console.error('Error: ${err}');
				}
			}

			function isAtBottom(elem) {
				return elem.scrollTop >= (elem.scrollHeight - elem.offsetHeight);
			}
			function scrollBottom(elem) {
				elem.scrollTop = elem.scrollHeight;
			}
		}

	</script>
	</head>

	Command: ` + logCmdStr + `</br>
	<form id="restartForm">
	<input type="password" id="pw" name="pw" placeholder="password">
	<input type="button" id="restart-btn" value="Restart"/></br></br>
	</form>
	<div id="log_text" />
	</body>
	</html>
	`
	w.Write([]byte(body))
}

func restartResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		//fmt.Printf("%+v", r)
		//r.ParseForm()
		pw := r.FormValue("pw")

		if viper.GetString("Password") != pw {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Printf("Unauthorized restart attempt using pw=%v\n", pw)
			return
		}
	}
	w.WriteHeader(http.StatusOK)

	killRunningCommand()
	runCommand()
}

func logResponse(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(logtext))
}

func getLogCmdStr(cmd string, args []string) string {
	logCmdStr := cmd
	if len(args) > 0 {
		for _, arg := range args {
			if strings.ContainsAny(arg, " ") {
				logCmdStr += " '" + arg + "'"
			} else {
				logCmdStr += " " + arg
			}
		}
	}
	return logCmdStr
}

func killRunningCommand() {
	if cmd != nil {
		cmd.Process.Kill()
		cmd.Wait()
		// append a horizontal line to the log for connected clients, then clear it for any future connections
		appendToLog("<hr>")
		clearLog()
	}
}

func runCommand() {
	cmdStr := viper.GetString("Command")
	argArray := viper.GetStringSlice("Args")

	appendToLog("Starting Command `" + getLogCmdStr(cmdStr, argArray) + "` at " + time.Now().String() + "\n")

	if len(argArray) > 0 {
		cmd = exec.Command(cmdStr, argArray...)
	} else {
		cmd = exec.Command(cmdStr)
	}
	so, _ := cmd.StdoutPipe()
	se, _ := cmd.StderrPipe()
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	stdout = bufio.NewScanner(so)
	stderr = bufio.NewScanner(se)
}

func clearLog() {
	logtext = ""
}

func appendToLog(s string) {
	s = strings.Replace(s, "\n", "<br>", -1)
	logtext += s

	if melodyWs != nil {
		melodyWs.Broadcast([]byte(s))
	}
}

func readAllStdout() {
	for {
		if stdout != nil {
			for stdout.Scan() {
				m := stdout.Text()
				if viper.GetBool("StdoutToTerminal") {
					fmt.Println(m)
				}
				appendToLog(m + "\n")
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func readAllStderr() {
	for {
		if stderr != nil {
			for stderr.Scan() {
				m := stderr.Text()
				if viper.GetBool("StderrToTerminal") {
					fmt.Println(m)
				}
				appendToLog(m + "\n")
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	viper.SetDefault("ServerAddress", ":8080")
	viper.SetDefault("Command", "ls")
	viper.SetDefault("Args", "-lh")
	viper.SetDefault("RunOnLaunch", false)
	viper.SetDefault("NoWebsocketOriginCheck", false)
	viper.SetDefault("StdoutToTerminal", true)
	viper.SetDefault("StderrToTerminal", true)
	viper.SetDefault("Password", "")

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("$HOME/.cmdwebctrl")
	viper.AddConfigPath(".")

	viper.WatchConfig()

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	// start the goroutine for updating log file text
	go readAllStdout()
	go readAllStderr()

	if viper.GetBool("RunOnLaunch") {
		runCommand()
	}

	// Start melody websocket framework
	melodyWs = melody.New()

	// Allow all origin hosts if needed
	if viper.GetBool("NoWebsocketOriginCheck") {
		melodyWs.Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	}

	// configure endpoints and start the web server
	http.HandleFunc("/", mainPageResponse)
	http.HandleFunc("/restart", restartResponse)
	http.HandleFunc("/log", logResponse)
	http.HandleFunc("/logws", func(w http.ResponseWriter, r *http.Request) {
		melodyWs.HandleRequest(w, r)
	})

	melodyWs.HandleConnect(func(s *melody.Session) {
		s.Write([]byte(logtext))
	})

	log.Fatal(http.ListenAndServe(viper.GetString("ServerAddress"), nil))
}
