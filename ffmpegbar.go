package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func humanReadableClock(unixTime int64) string {
	return time.Unix(unixTime, 0).Format("15:04:05 PM")
}

func humanReadableTime(unixTime int64) string {
	newTime := time.Unix(unixTime, 0)
	_, offset := newTime.Zone()
	// add the offset to the time
	newTime = newTime.Add(time.Duration(-offset) * time.Second)
	return newTime.Format("15:04:05")
}

func humanReadableSize(size int64, baseUnit string, decimalPlaces int64) string {
	new_size := float64(size)
	var unit string
	for _, unit = range []string{"", "K", "M", "G", "T", "P", "E", "Z", "Y"} {
		if new_size < 1024.0 {
			break
		}
		new_size /= 1024.0
	}

	return fmt.Sprintf("%.*f %s%s", decimalPlaces, new_size, unit, baseUnit)
}

func progressBar(value int64, endvalue int64, startTime int64, barLength int64) string {
	if endvalue == 0 {
		return ""
	}
	if barLength == 0 {
		barLength = 50
	}
	percent := (float64(value) / float64(endvalue)) * 100

	bar_fill := int64((float64(barLength) / 100) * percent)
	bar_empty := barLength - bar_fill
	bar := strings.Repeat("=", int(bar_fill)) + strings.Repeat(" ", int(bar_empty))

	time_taken := time.Now().Unix() - startTime
	time_left := int64(float64(time_taken) / (percent / 100))
	eta := startTime + time_left
	return fmt.Sprintf("%d / %d [%s] %d%% %s / %s (%s) %s", value, endvalue, bar, int(percent), humanReadableTime(time_taken), humanReadableTime(time_left), humanReadableClock(eta), strings.Repeat(" ", 10))

}

type MediaProp struct {
	framerate float64
	frames    int64
	widht     float64
	height    float64
	duration  int64
	bitrate   int64
	size      int64
}

func getMediaProps(path string) (MediaProp, error) {
	// use ffprobe to get media properties: width, height, r_frame_rate, duration, bitrate
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout
	output, err := cmd.Output()
	if err != nil {
		return MediaProp{}, err
	}
	var data map[string]interface{}
	if err = json.Unmarshal(output, &data); err != nil {
		return MediaProp{}, err
	}
	stream := data["streams"].([]interface{})[0].(map[string]interface{})
	framerate := stream["r_frame_rate"].(string)
	var framerate_flt float64
	// if framerate contains a slash, evaluate it then assign it to framerate_flt, otherwise just parse it as float
	if strings.Contains(framerate, "/") {
		raw := strings.Split(framerate, "/")
		A, _ := strconv.ParseFloat(raw[0], 64)
		B, _ := strconv.ParseFloat(raw[1], 64)
		framerate_flt = A / B
	} else {
		framerate_flt, _ = strconv.ParseFloat(framerate, 64)
	}
	frames, _ := strconv.ParseInt(stream["nb_frames"].(string), 10, 64)
	width := stream["width"].(float64)
	height := stream["height"].(float64)
	duration, _ := strconv.ParseFloat(stream["duration"].(string), 64)
	bitrate, _ := strconv.ParseInt(stream["bit_rate"].(string), 10, 64)
	size, _ := strconv.ParseInt(data["format"].(map[string]interface{})["size"].(string), 10, 64)
	fmt.Println()
	return MediaProp{framerate_flt, frames, width, height, int64(duration), bitrate, size}, nil
}

func parseFfmpegStatus(stdout string) (int64, float64) {

	if len(stdout) == 0 || !strings.HasPrefix(stdout, "frame=") {
		return 0, 0
	}

	frame := strings.Split(stdout, "frame=")[1]
	frame = strings.Split(frame, "fps=")[0]
	frame = strings.TrimSpace(frame)
	frame_int, _ := strconv.ParseInt(frame, 10, 64)
	fps := strings.Split(stdout, "fps=")[1]
	fps = strings.Split(fps, "q=")[0]
	fps = strings.TrimSpace(fps)
	fps_float, _ := strconv.ParseFloat(fps, 64)

	return frame_int, fps_float
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("No argument")
		os.Exit(1)
	}
	ffmpeg_params := os.Args[1:]

	// Check output file exists
	output_file := ffmpeg_params[len(ffmpeg_params)-1]
	if _, err := os.Stat(output_file); err == nil {
		fmt.Println("Output file already exists")
		os.Exit(1)
	}

	// Check input file exists
	var input_index int
	for i, v := range ffmpeg_params {
		if v == "-i" {
			input_index = i
			break
		}
	}
	input_file := ffmpeg_params[input_index+1]
	if _, err := os.Stat(input_file); err != nil {
		fmt.Println("Input file does not exist")
		os.Exit(1)
	}

	// Get media properties
	media, err := getMediaProps(input_file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	startTime := time.Now().Unix()

	// Start ffmpeg
	cmd := exec.Command("ffmpeg", ffmpeg_params...)
	ffmpegOutput, _ := cmd.StderrPipe()
	cmd.Start()
	stdout := bufio.NewReader(ffmpegOutput)
	for {
		out, _ := stdout.ReadString('\r')
		if len(out) == 0 {
			break
		}
		frame, fps := parseFfmpegStatus(out)
		fmt.Printf("\r%.2f fps %s", fps, progressBar(frame, media.frames, startTime, 20))
	}
	fmt.Printf("\r%.2ffps %s", 0.0, progressBar(media.frames, media.frames, startTime, 20))
	cmd.Wait()

	// if output file is not /dev/null
	if output_file != "/dev/null" {
		media, err = getMediaProps(output_file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("\n==> Output file:", ffmpeg_params[len(ffmpeg_params)-1])
		fmt.Println("- Resolution:", media.widht, "x", media.height)
		fmt.Printf("- Framerate: %.2ffps\n", media.framerate)
		fmt.Println("- Duration:", humanReadableTime(media.duration))
		fmt.Println("- Bitrate:", humanReadableSize(media.bitrate, "b", 2))
		fmt.Println("- Size:", humanReadableSize(media.size, "B", 2))
	}
}
