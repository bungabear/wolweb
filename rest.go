// Rest API Implementations

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// restWakeUpWithDeviceName - REST Handler for Processing URLS /virtualdirectory/apipath/<deviceName>
func wakeUpWithDeviceName(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	deviceName := vars["deviceName"]

	var result HTTPResponseObject
	result.Success = false

	// Ensure deviceName is not empty
	if deviceName == "" {
		// Devicename is empty
		result.Message = "Empty device names are not allowed."
		result.ErrorObject = nil
		w.WriteHeader(http.StatusBadRequest)
	} else {

		// Get Device from List
		for _, c := range appData.Devices {
			if c.Name == deviceName {

				// We found the Devicename
				if err := SendMagicPacket(c.Mac, c.BroadcastIP, c.Interface); err != nil {
					// We got an internal Error on SendMagicPacket
					w.WriteHeader(http.StatusInternalServerError)
					result.Success = false
					result.Message = "An internal error occurred while sending the Magic Packet."
					result.ErrorObject = err
				} else {
					// Horray we send the WOL Packet succesfully
					result.Success = true
					result.Message = fmt.Sprintf("Sent magic packet to device '%s' with MAC '%s' on Broadcast IP '%s' with interface '%s'.", c.Name, c.Mac, c.BroadcastIP, c.Interface)
					result.ErrorObject = nil
				}
			}
		}

		if !result.Success && result.ErrorObject == nil {
			// We could not find the Devicename
			w.WriteHeader(http.StatusNotFound)
			result.Message = fmt.Sprintf("Device name '%s' could not be found.", deviceName)
		}
	}

	json.NewEncoder(w).Encode(result)
}

// pingDeviceByName - REST Handler for Processing URLS /virtualdirectory/ping/<deviceName>
func pingDeviceByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	deviceName := vars["deviceName"]

	var result HTTPResponseObject
	result.Success = false

	if deviceName == "" {
		result.Message = "Empty device names are not allowed."
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(result)
		return
	}

	var device *Device
	for i := range appData.Devices {
		if appData.Devices[i].Name == deviceName {
			device = &appData.Devices[i]
			break
		}
	}

	if device == nil {
		w.WriteHeader(http.StatusNotFound)
		result.Message = fmt.Sprintf("Device name '%s' could not be found.", deviceName)
		json.NewEncoder(w).Encode(result)
		return
	}

	targetIP := strings.TrimSpace(device.TargetIP)
	if targetIP == "" {
		w.WriteHeader(http.StatusBadRequest)
		result.Message = fmt.Sprintf("Device '%s' has no IP configured for ping.", deviceName)
		json.NewEncoder(w).Encode(result)
		return
	}

	output, err := runPing(targetIP)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		result.Success = false
		result.ErrorObject = err
		result.Message = fmt.Sprintf("Ping to '%s' failed. %s", targetIP, output)
		json.NewEncoder(w).Encode(result)
		return
	}

	result.Success = true
	result.Message = fmt.Sprintf("Ping to '%s' succeeded.", targetIP)
	json.NewEncoder(w).Encode(result)
}

func runPing(targetIP string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "ping", "-n", "1", "-w", "1500", targetIP)
	case "darwin":
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "2000", targetIP)
	default:
		cmd = exec.CommandContext(ctx, "ping", "-c", "1", "-W", "2", targetIP)
	}

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if output == "" {
		output = "No output"
	}

	if ctx.Err() == context.DeadlineExceeded {
		return "Ping timed out", ctx.Err()
	}

	return output, err
}
