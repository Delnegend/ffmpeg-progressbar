import subprocess as sp
import time
import sys
import json
import os
# import re


def humanReadableTime(seconds):
    hour = int(seconds / 3600)
    minute = int((seconds % 3600) / 60)
    second = int(seconds % 60)
    return f'{hour:02d}:{minute:02d}:{second:02d}'

def humanReadableSize(size, decimal_places=2):
    for unit in ['','K','M','G','T','P','E','Z']:
        if size < 1024.0:
            break
        size /= 1024.0
    return f"{size:.{decimal_places}f} {unit}B"

def progressBar(value, endvalue, start_time, bar_length=20):
    percent = float(value) / endvalue if endvalue else 0
    bar = '[' + '=' * int(round(percent * bar_length) - 1) + '>' + ' ' * (
        bar_length - int(round(percent * bar_length))) + ']'
    time_taken = time.time() - start_time
    eta = (time_taken / value) * (endvalue - value) if value else 0
    finish_at = time.localtime(time.time() + eta)
    return f'{bar} {percent*100:.2f}% {humanReadableTime(time_taken)} / {humanReadableTime(eta)} ({time.strftime("%I:%M:%S %p", finish_at)})'


def getMediaProperties(path):
    proc = sp.Popen(
        "ffprobe -select_streams v:0 -v quiet -print_format json -show_format -show_streams -show_error"
        .split() + [path],
        stdout=sp.PIPE,
        stderr=sp.PIPE)
    out, err = proc.communicate()
    return json.loads(out)["streams"][0]


def parseFfmpegStatus(stdout):
    data = stdout.readline()
    # if the line is empty or not starting with frame=, skip it
    if not data or not data.startswith('frame='):
        return None

    # ==============================================
    # GETTING EVERYTHING FROM FFMPEG PROGRESS OUTPUT
    # ==============================================

    # tidy up the line:
    # - replace multiple spaces with one, remove trailing spaces
    # - split the line with "=" as separator
    # => ['frame', '12 fps', '12 q', '12.3 size', '1234kB time', '12:34:56.78 bitrate', '1234.5kbits/s speed', '1.234x']

    # line = [f.strip() for f in re.sub(' +', ' ', data).split('=')]

    # split each element of the line into a list of two elements if it contains a space
    # => ['frame', '12', 'fps', '12', 'q', '12.3', 'size', '1234kB', 'time', '12:34:56.78', 'bitrate', '1234.5kbits/s', 'speed', '1.234x']

    # for i, f in enumerate(line):
    #     if ' ' in f:
    #         line[i:i + 1] = f.split(' ')
    # data = dict(zip(line[::2], line[1::2]))

    # =====================================
    # JUST GETTING THE FRAME NUMBER AND FPS
    # =====================================

    frame = data.split('frame=')[1].split('fps=')[0].strip()
    fps = data.split('fps=')[1].split('q=')[0].strip()
    return {'frame': frame, 'fps': fps}


def main():

    ffmpeg_params = sys.argv[1:]
    total_frames = int(
        getMediaProperties(ffmpeg_params[ffmpeg_params.index('-i') +
                                         1])['nb_frames'])
    start_time = time.time()
    proc = sp.Popen(['ffmpeg'] + ffmpeg_params,
                    stdout=sp.PIPE,
                    stderr=sp.PIPE,
                    universal_newlines=True)
    while proc.poll() is None:
        data = parseFfmpegStatus(proc.stderr)
        if data is not None:
            print(
                f"{data['frame']} / {total_frames} {data['fps']} {progressBar(int(data['frame']), total_frames, start_time)}",
                end='\r')
        time.sleep(0.1)
    print(
        f"{total_frames} / {total_frames} {data['fps']}fps {progressBar(total_frames, total_frames, start_time)}"
    )

    print()
    output_file = ffmpeg_params[-1]
    data = getMediaProperties(output_file)
    resolution = f"{data['width']}x{data['height']}"
    frame_rate = round(int(data['r_frame_rate'].split('/')[0]) / int(data['r_frame_rate'].split('/')[1]), 2) if '/' in data['r_frame_rate'] else data['r_frame_rate']
    bitrate = f"{round(int(data['bit_rate'])/(10**6), 2)} Mbps"
    out_size = humanReadableSize(os.path.getsize(output_file))
    print(f"==> Output file: {output_file}")
    print(f"- Resolution: {resolution}")
    print(f"- Frame rate: {frame_rate} fps")
    print(f"- Bitrate: {bitrate}")
    print(f"- Size: {out_size}")

if __name__ == '__main__':
    main()