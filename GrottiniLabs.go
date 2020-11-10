package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey"
	"gopkg.in/ini.v1"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func main() {

	type apiResponseStruct struct {
		Status   int      `json:"status"`
		Error    string   `json:"error"`
		Response []string `json:"response"`
	}
	ApiResponse := apiResponseStruct{}

	/*if runtime.GOOS == "windows" {
		fmt.Println("Can't Execute this on a windows machine")
		return
	}*/
	fmt.Println("-------------------------------------------------------------------------------------------")
	fmt.Println("|	Welcome to grottinilab Tools installer wizard.")
	fmt.Println("|	This tool will allow you easily install your desired software on the machine quickly.")
	fmt.Println("-------------------------------------------------------------------------------------------")
	fmt.Println("")

	// Answers to the questions are stored in these structs =================================
	answerSoftware := struct {
		SType string `survey:"softwareType"` // or you can tag fields to match a specific name
	}{}
	answerSystemVersion := struct {
		SystemVersion string `survey:"systemVersion"` // or you can tag fields to match a specific name
	}{}
	answerSoftwareVersion := struct {
		SystemVersion string `survey:"softwareVersion"` // or you can tag fields to match a specific name
	}{}
	questionIpAnswer := struct {
		IpAddress string
	}{}

	//minute and second when machine will be restarted
	questionCrontabTime := struct {
		hour string
		minute string
	}{}


	//do we install controller system?
	questionControllerSystem := struct {
		InstallControllerSystem bool
	}{}

	//if so, specify the location
	questionControllerSystemlocDescription := struct {
		LocationDescription     string
	}{}

	//these questions are asked when you install controller system solely
	questioncontrollersystemCs := struct {
		Description		string
		LocationName    string
		SoftwareName	string
		LocationPath	string
	}{}

	// request master id, where master id is a string
	questionRequestMasterID := struct {
		MasterID	string
	}{}

	//request location id and camera id
	questionRequestCameraLocationID := struct {
		CameraID	int
		LocationID	int
	}{}
	//=======================================================================================

	//start off by fetching the list of available software(s)
	Resp, err1 := http.Get("http://server:8282/api/getsoftwaretypes?token=S3pmbe01FAHaEgnG")
	if err1 != nil {
		print(err1)
	}
	defer Resp.Body.Close()
	json.NewDecoder(Resp.Body).Decode(&ApiResponse)
	if err1 != nil {
		print(err1)
	}
	var answersSoftwareList []string
	for _, element := range ApiResponse.Response {
		answersSoftwareList = append(answersSoftwareList, strings.Replace(element, "_", " ", -1))
	}
	answersSoftwareList = append(answersSoftwareList, "Controller System", "Exit Wizard...")

	// define and prepare the question to ask
	var qs = []*survey.Question{
		{
			Name: "softwareType",
			Prompt: &survey.Select{
				Message: "Which software would you like to choose?:",
				Options: answersSoftwareList,
			},
			Validate: survey.Required,
		},
	}

	// perform the question
	err := survey.Ask(qs, &answerSoftware)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("%s",answerSoftware.SType)
	if answerSoftware.SType == "Exit Wizard..." { //user asked to leave the wizard
		os.Exit(1)
	} else if answerSoftware.SType == "Controller System" { // user asked to install only controller system

		var qcontrollersystemCs = []*survey.Question{
			{
				Name:   "Description",
				Prompt:   &survey.Input{Message: "Please enter description:"},
				Validate: survey.Required,
			},
			{
				Name:   "LocationName",
				Prompt: &survey.Input{Message: "Please enter location name:"},
				Validate: survey.Required,
			},
			{
				Name:   "SoftwareName",
				Prompt: &survey.Input{Message: "Please enter software name:"},
				Validate: survey.Required,
			},
			{
				Name:   "LocationPath",
				Prompt: &survey.Input{Message: "Please enter software location path:"},
				Validate: survey.Required,
			},
		}
		errCs := survey.Ask(qcontrollersystemCs, &questioncontrollersystemCs)
		if errCs != nil {
			fmt.Println(errCs.Error())
			return
		}

		myStr := "http://server:8282/api/getcs?token=S3pmbe01FAHaEgnG"
		responseGetCs, _ := http.Get(myStr)
		fmt.Println("Please wait while the Controller system is being downloaded...")
		exec.Command("mkdir", "/home/grottinilab/develop").Output()
		exec.Command("mkdir", "/home/grottinilab/develop/repository").Output()
		filepathCs, _ := os.Create("/opt/ControllerSystem.7z")
		defer filepathCs.Close()
		defer responseGetCs.Body.Close()
		io.Copy(filepathCs, responseGetCs.Body)
		fmt.Println("Controller system has been downloaded successfully, installing...")
		exec.Command("sudo", "apt-get", "install", "p7zip-full").Output()
		exec.Command("7z", "x", "/opt/ControllerSystem.7z", "-o/opt/").Output()
		exec.Command("sudo", "apt-get", "install", "-y", "curl").Output()
		installNodeCommand := "curl -sL https://deb.nodesource.com/setup_8.x | sudo -E bash -"
		exec.Command("bash", "-c", installNodeCommand).Output()
		exec.Command("sudo", "apt-get", "install", "-y", "nodejs").Output()

		cfg, err := ini.Load("/opt/controllerSystem/config.ini")
		if err != nil {
			fmt.Printf("Fail to read file: %v", err)
			os.Exit(1)
		}
		cfg.Section("application").Key("description").SetValue(questioncontrollersystemCs.Description)
		cfg.Section("application").Key("location").SetValue(questioncontrollersystemCs.LocationName)
		cfg.Section("application").Key("name").SetValue(questioncontrollersystemCs.SoftwareName)
		cfg.Section("application").Key("directory").SetValue(questioncontrollersystemCs.LocationPath)
		cfg.SaveTo("/opt/controllerSystem/config.ini")
		exec.Command("sudo", "cp", "/opt/controllerSystem/controller-system.service", "/lib/systemd/system/").Output()
		exec.Command("sudo", "systemctl", "start", "controller-system").Output()
		exec.Command("sudo", "systemctl", "enable", "controller-system").Output()
		fmt.Println("Controller system has been successfully installed!")
		exec.Command("sudo", "rm", "/opt/ControllerSystem.7z").Output()

	} else { // user asked to install one of the software

		//fetch the available system versions and ask the user to choose
		myStr := "http://server:8282/api/getsystemversion?software=" + strings.Replace(answerSoftware.SType, " ", "_", -1) + "&token=S3pmbe01FAHaEgnG"
		Resp_, _ := http.Get(myStr)
		defer Resp_.Body.Close()
		json.NewDecoder(Resp_.Body).Decode(&ApiResponse)
		var answersVersionList []string
		for _, element := range ApiResponse.Response {
			answersVersionList = append(answersVersionList, strings.Replace(element, "_", " ", -1))
		}
		// define and prepare the question
		var qsSystemVersion = []*survey.Question{
			{
				Name: "systemVersion",
				Prompt: &survey.Select{
					Message: "What's the system version?:",
					Options: answersVersionList,
				},
				Validate: survey.Required,
			},
		}

		//perform the question
		errSystemVersionQs := survey.Ask(qsSystemVersion, &answerSystemVersion)
		if errSystemVersionQs != nil {
			fmt.Println(errSystemVersionQs.Error())
			return
		}


		//fetch the available system versions and ask the user to choose
		myStr = "http://server:8282/api/getsoftwareversion?software=" + strings.Replace(answerSoftware.SType, " ", "_", -1) + "&systemversion=" + strings.Replace(answerSystemVersion.SystemVersion, " ", "_", -1) + "&token=S3pmbe01FAHaEgnG"
		responseGetSoftwareVersion, _ := http.Get(myStr)
		defer responseGetSoftwareVersion.Body.Close()
		json.NewDecoder(responseGetSoftwareVersion.Body).Decode(&ApiResponse)
		var answersSoftwareVersionList []string
		for _, element := range ApiResponse.Response {
			answersSoftwareVersionList = append(answersSoftwareVersionList, strings.Replace(element, "_", " ", -1))
		}
		// define and prepare the question
		var qsSoftwareVersion = []*survey.Question{
			{
				Name: "softwareVersion",
				Prompt: &survey.Select{
					Message: "What's the software version?:",
					Options: answersSoftwareVersionList,
				},
				Validate: survey.Required,
			},
		}

		//perform the question
		errSoftwareVersionQs := survey.Ask(qsSoftwareVersion, &answerSoftwareVersion)
		if errSoftwareVersionQs != nil {
			fmt.Println(errSoftwareVersionQs.Error())
			return
		}

		//finally fetch the file we need and download it
		myStr = "http://server:8282/api/getsoftware?software=" + strings.Replace(answerSoftware.SType, " ", "_", -1) + "&systemversion=" + strings.Replace(answerSystemVersion.SystemVersion, " ", "_", -1) + "&softwareversion=" + answerSoftwareVersion.SystemVersion + "&token=S3pmbe01FAHaEgnG"
		responseGetSoftware, _ := http.Get(myStr)
		getContentLength := responseGetSoftware.Header["Content-Length"]
		if  getContentLength[0] == "5"{
			fmt.Printf("No file was returned by the server for the selected software and version, please choose another software!\n")
			os.Exit(1)
		}
		fmt.Println("Please wait while the software is being downloaded...")
		exec.Command("sudo", "apt-get", "install", "p7zip-full").Output()
		exec.Command("mkdir", "/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"").Output()
		filepath, _ := os.Create("/opt/" + strings.Replace(answerSoftware.SType, " ", "_", -1) + "/software.7z")
		defer filepath.Close()
		defer responseGetSoftware.Body.Close()
		io.Copy(filepath, responseGetSoftware.Body)
		exec.Command("7z", "x", "/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/software.7z", "-o/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"").Output()
		fmt.Println("Software has been downloaded successfully!")
		exec.Command("sudo", "rm", "/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/software.7z").Output()

		//ask if we want to install team viewer on the machine ----------------
		installTeamViewer := false
		prompt := &survey.Confirm{
			Message: "Install teamviewer?",
		}
		survey.AskOne(prompt, &installTeamViewer)
		if installTeamViewer == true {
			fmt.Printf("Installing teamviewer...\n")

			exec.Command("mkdir", "temp").Output()
			exec.Command("sudo", "wget", "http://download.teamviewer.com/download/version_12x/teamviewer_i386.deb", "-O", "temp/teamviewer.deb").Output()
			exec.Command("sudo", "apt", "install", "-f", "-y", "./temp/teamviewer.deb").Output()
			exec.Command("teamviewer").Output()
			exec.Command("rm", "-rf", "temp").Output()
		} else {
			fmt.Println("Skipping teamviewer installation...")
		}

		//----------------------------------------------------------------------

		//ask if wifi should be disabled or not --------------------------------
		wifiStatus := false
		prompt = &survey.Confirm{
			Message: "Turn wifi off?",
		}
		survey.AskOne(prompt, &wifiStatus)
		if wifiStatus == true {
			exec.Command("nmcli", "radio", "wifi", "off").Output()
		} else {
			fmt.Println("Keeping wifi on")
		}
		//-----------------------------------------------------------------------

		//ask for the ip address-------------------------------------------------
		var questionIp = []*survey.Question{
			{
				Name:      "IpAddress",
				Prompt:    &survey.Input{Message: "Please enter the machine IP address:"},
				Validate:  survey.Required,
				Transform: survey.Title,
			},
		}
		err = survey.Ask(questionIp, &questionIpAnswer)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		//------------------------------------------------------------------------

		//should we install controller system?------------------------------------
		var qControllerSystemInstall = []*survey.Question{
			{
				Name:     "installControllerSystem",
				Prompt:   &survey.Confirm{Message: "Install controller system?"},
				Validate: survey.Required,
			},
		}
		errCs := survey.Ask(qControllerSystemInstall, &questionControllerSystem)
		if errCs != nil {
			fmt.Println(errCs.Error())
			return
		}

		if questionControllerSystem.InstallControllerSystem == true {

			var qControllerSystemlocDescription = []*survey.Question{
				{
					Name:     "locationDescription",
					Prompt:   &survey.Input{Message: "Please enter location description:"},
					Validate: survey.Required,
				},
			}
			errCs := survey.Ask(qControllerSystemlocDescription, &questionControllerSystemlocDescription)
			if errCs != nil {
				fmt.Println(errCs.Error())
				return
			}

			myStr = "http://server:8282/api/getcs?token=S3pmbe01FAHaEgnG"
			responseGetCs, _ := http.Get(myStr)
			fmt.Println("Please wait while the Controller system is being downloaded...")
			filepathCs, _ := os.Create("/opt/ControllerSystem.7z")
			defer filepathCs.Close()
			defer responseGetCs.Body.Close()
			io.Copy(filepathCs, responseGetCs.Body)
			fmt.Println("Controller system has been downloaded successfully, installing...")
			exec.Command("sudo", "apt-get", "install", "p7zip-full").Output()
			exec.Command("7z", "x", "/opt/ControllerSystem.7z", "-o/opt/").Output()


			error, output := exec.Command("sudo", "apt-get", "install", "-y", "curl").Output()
			if error != nil {
				log.Fatal(error)
			}
			fmt.Printf("%s\n", output)
			error, output =exec.Command("curl", "-sL", "https://deb.nodesource.com/setup_8.x", "|", "sudo", "-E", "bash", "-").Output()
			if error != nil {
				log.Fatal(error)
			}
			fmt.Printf("%s\n", output)
			error, output =exec.Command("sudo", "apt-get", "install", "-y", "nodejs").Output()
			if error != nil {
				log.Fatal(error)
			}
			fmt.Printf("%s\n", output)


			exec.Command("sudo", "cp", "/opt/controllerSystem/controller-system.service", "/lib/systemd/system/").Output()
			exec.Command("sudo", "systemctl", "start", "controller-system").Output()
			exec.Command("sudo", "systemctl", "enable", "controller-system").Output()
			cfg, err := ini.Load("/opt/controllerSystem/config.ini")
			if err != nil {
				fmt.Printf("Fail to read file: %v", err)
				os.Exit(1)
			}
			cfg.Section("application").Key("description").SetValue(answerSoftware.SType)
			cfg.Section("application").Key("location").SetValue(questionControllerSystemlocDescription.LocationDescription)
			cfg.Section("application").Key("name").SetValue(strings.Replace(answerSoftware.SType, " ", "_", -1))
			cfg.Section("application").Key("directory").SetValue("/opt/" + strings.Replace(answerSoftware.SType, " ", "_", -1) + "")
			cfg.SaveTo("/opt/controllerSystem/config.ini")
			fmt.Println("Controller system has been successfully installed!")
			exec.Command("sudo", "rm", "/opt/ControllerSystem.7z").Output()
		}

		fmt.Printf("%s is now being installed on this machine... please wait patiently.\n", answerSoftware.SType)

		//install ubuntu updates
		fmt.Printf("Updating ubuntu system...\n")
		exec.Command("sudo", "apt-get", "-y", "update").Output()

		//install required libraries
		fmt.Print("Installing required libraries...\n")
		exec.Command("sudo", "apt-get", "install", "-y", "htop", "openssh-server", "dconf-editor", "python-pip", "p7zip-full", "ffmpeg").Output()

		//Enable Desktop Sharing
		fmt.Print("Enabling Desktop Sharing...\n")
		exec.Command("dconf", "write", "/org/gnome/desktop/remote-access/require-encryption", "false").Output()
		exec.Command("gsettings", "set", "org.gnome.Vino", "enabled", "true").Output()
		exec.Command("gsettings", "set", "org.gnome.Vino", "prompt-enabled", "false").Output()


		//Disable auto Updates
		fmt.Print("Disabling auto updates...\n")
		exec.Command("gsettings", "set", "com.ubuntu.update-notifier", "no-show-notifications", "true").Output()
		exec.Command("sudo", "sed", "-i", "'s/APT::Periodic::Update-Package-Lists \"1\"/APT::Periodic::Update-Package-Lists \"0\"/'", "/etc/apt/apt.conf.d/20-auto-upgrades").Output()

		//Screen
		fmt.Print("Setting screen settings...\n")
		exec.Command("gsettings", "set", "org.gnome.desktop.session", "idle-delay", "0").Output()
		exec.Command("gsettings", "set", "org.gnome.desktop.screensaver", "lock-enabled", "false").Output()
		exec.Command("gsettings", "set", "org.gnome.desktop.lockdown", "disable-lock-screen", "true").Output()

		//Install software requirements
		fmt.Print("Installing software requirements...\n")
		exec.Command("python", "-m", "pip", "install", "numpy", "pyserial", "pillow", "boto3").Output()
		exec.Command("sudo", "adduser", "grottinilab", "dialout").Output()

		//Recordfail Timeout
		fmt.Print("setting recordfail timeout...\n")
		exec.Command("sudo", "sed", "-i", "'s/^GRUB_TIMEOUT.*/& \nGRUB_RECORDFAIL_TIMEOUT=0/'", "/etc/default/grub").Output()
		exec.Command("sudo", "update-grub").Output()


		//ask the time when the machine will be restarted through crontab------------------------------------
		var qCrontabTime = []*survey.Question{
			{
				Name:     "hour",
				Prompt:   &survey.Input{Message: "Install controller system?"},
				Validate: func (val interface{}) error {
					str, ok := val.(string)
					if  _, err := strconv.Atoi(str); !ok || err != nil {
						return errors.New("Please enter a number")
					}
					return nil
				},
			},
			{
				Name:     "minute",
				Prompt:   &survey.Input{Message: "Install controller system?"},
				Validate: func (val interface{}) error {
					str, ok := val.(string)
					if  _, err := strconv.Atoi(str); !ok || err != nil {
						return errors.New("Please enter a number")
					}
					return nil
				},
			},
		}
		errCrontabTime := survey.Ask(qCrontabTime, &questionCrontabTime)
		if errCrontabTime != nil {
			fmt.Println(errCrontabTime.Error())
			return
		}
		f, err := os.Create("glabcrontab")
		if err != nil {
			fmt.Println(err)
			return
		}
		f.WriteString(""+questionCrontabTime.minute+" "+questionCrontabTime.hour+" * * * reboot")
		exec.Command("sudo", "mv", "glabcrontab", "/etc/cron.d").Output()

		if answerSoftware.SType == "people counter front" || answerSoftware.SType == "people counter top" || answerSoftware.SType == "people counter top trigger" || answerSoftware.SType == "shopper analytics asus" || answerSoftware.SType == "shopper analytics intel" || answerSoftware.SType == "xovis tcp service"{

			var qRequestCameraLocationID = []*survey.Question{
				{
					Name:     "LocationID",
					Prompt:   &survey.Input{Message: "Please enter the Location ID:"},
					Validate: func (val interface{}) error {
						str, ok := val.(string)
						if  _, err := strconv.Atoi(str); !ok || err != nil {
							return errors.New("Please enter a number")
						}
						return nil
					},
				},
				{
					Name:     "CameraID",
					Prompt:   &survey.Input{Message: "Please enter the Camera ID:"},
					Validate: func (val interface{}) error {
						str, ok := val.(string)
						if  _, err := strconv.Atoi(str); !ok || err != nil {
							return errors.New("Please enter a number")
						}
						return nil
					},
				},
			}
			errCs := survey.Ask(qRequestCameraLocationID, &questionRequestCameraLocationID)
			if errCs != nil {
				fmt.Println(errCs.Error())
				return
			}
			cfg, err := ini.Load("/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/config.ini")
			if err != nil {
				fmt.Printf("Fail to read file: %v", err)
				os.Exit(1)
			}
			locationIDSave := strconv.Itoa(questionRequestCameraLocationID.LocationID)
			cameraIDSave := strconv.Itoa(questionRequestCameraLocationID.CameraID)
			cfg.Section("shop").Key("id").SetValue(locationIDSave)
			cfg.Section("camera").Key("id").SetValue(cameraIDSave)
			cfg.SaveTo("/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/config.ini")

			//write into /etc/network/interfaces
			offsetToSave := strconv.Itoa(questionRequestCameraLocationID.CameraID-10)
			valueToWrite := []byte("auto lo\niface lo inet loopback\nauto ip link | awk -F: '$0 !~ \"lo|vir|wl|^[^0-9]\"{print $2a;getline}'\niface ip link | awk -F: '$0 !~ \"lo|vir|wl|^[^0-9]\"{print $2a;getline}' inet static\naddress 192.168.0.$(("+cameraIDSave+"-"+(offsetToSave)+"))\nnetmask 255.255.255.0\ngateway 192.168.0.1\ndns-nameservers 192.168.0.1")
			ioutil.WriteFile("/etc/network/interfaces", valueToWrite, 0644 )

			//write into /etc/hostname
			valueToWrite = []byte("camera"+cameraIDSave+"")
			ioutil.WriteFile("/etc/hostname", valueToWrite, 0644 )

			//write into /etc/hosts
			valueToWrite = []byte("127.0.0.1 localhost\n127.0.1.1 camera"+cameraIDSave+"\n::1 ip6-localhost ip6-loopback\nfe00::0 ip6-localnet\nff00::0 ip6-mcastprefix\nff02::1 ip6-allnodes\nff02::2 ip6-allrouters")
			ioutil.WriteFile("/etc/hosts", valueToWrite, 0644 )

			if answerSoftware.SType == "shopper analytics asus"{
				exec.Command("sudo", "apt-get", "install", "g++").Output()
				exec.Command("sudo", "apt-get", "install", "python").Output()
				exec.Command("sudo", "apt-get", "install", "libusb-1.0-0-dev").Output()
				exec.Command("sudo", "apt-get", "install", "libudev-dev").Output()
				exec.Command("sudo", "apt-get", "install", "openjdk-8-jdk").Output()
				exec.Command("sudo", "apt-get", "install", "freeglut3-dev").Output()
				exec.Command("sudo", "apt-get", "install", "doxygen").Output()
				exec.Command("sudo", "apt-get", "install", "graphviz").Output()
				exec.Command("sudo", "apt-get", "install", "git").Output()
				exec.Command("git", "clone", "https://github.com/occipital/OpenNI2.git", "/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/").Output()
				exec.Command("/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/OpenNI2/Packaging/Linux/install.sh").Output()
				exec.Command("make", "-C", "/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/OpenNI2").Output()
				f, err := os.Create("/etc/ld.so.conf.d/openni.conf")
				if err != nil {
					fmt.Println(err)
					return
				}
				f.WriteString("/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/OpenNI2/bin/x64-release")
				exec.Command("sudo", "ldconfig").Output()
			}
			if answerSoftware.SType == "shopper analytics intel"{
				exec.Command("echo", "'deb http://realsense-hw-public.s3.amazonaws.com/Debian/apt-repo xenial main'", "|", "sudo", "tee", "/etc/apt/sources.list.d/realsense-public.list").Output()
				exec.Command("sudo", "apt-key", "adv", "--keyserver", "keys.gnupg.net", "--recv-key", "6F3EFCDE").Output()
				exec.Command("sudo", "apt-get", "install", "librealsense2-dkms").Output()
				exec.Command("sudo", "apt-get", "install", "librealsense2-utils").Output()
			}


		}else if answerSoftware.SType == "xovis udp service" {

			var qRequestMasterID = []*survey.Question{
				{
					Name:     "MasterID",
					Prompt:   &survey.Input{Message: "Please enter the Master ID [name]:"},
					Validate: survey.Required,
				},
			}
			errCs := survey.Ask(qRequestMasterID, &questionRequestMasterID)
			if errCs != nil {
				fmt.Println(errCs.Error())
				return
			}
			cfg, err := ini.Load("/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/config.ini")
			if err != nil {
				fmt.Printf("Fail to read file: %v", err)
				os.Exit(1)
			}
			cfg.Section("xovis").Key("master_id").SetValue(questionRequestMasterID.MasterID)
			cfg.SaveTo("/opt/"+strings.Replace(answerSoftware.SType, " ", "_", -1)+"/config.ini")

		}else {
			fmt.Printf("Software was not recognized by the installer, is it a new software?")
			fmt.Printf("No configuration settings will be made.")
		}
		fmt.Printf("%s has successfully been installed on the machine!\n",answerSoftware.SType)
		fmt.Printf("System will be rebooted in 10 seconds to complete the installation...")
		time.Sleep(10)
		exec.Command("reboot").Output()

	}

}

