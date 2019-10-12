#!/usr/bin/env python

# usage: vocoder.py [-h] [-t TRANSPOSE] [-f FORMANT] [-b BREATHINESS] [-e] [-v]
#                   in_file [out_file]
#
# positional arguments:
#   in_file               input wav file
#   out_file              output wav file (default: direct playback)
#
# optional arguments:
#   -h, --help            show this help message and exit
#   -t TRANSPOSE, --transpose TRANSPOSE
#                         transpose [semitones]
#   -c CORRECT_PITCH, --correct_pitch CORRECT_PITCH
#                         pitch correction [%]
#   -f FORMANT, --formant FORMANT
#                         formant shift [semitones]
#   -b BREATHINESS, --breathiness BREATHINESS
#                         breathiness boost [%]
#   -H, --high_quality    use Harvest instead of Dio
#   -v, --visualize       visualize f0, sp and ap with Sixel

import argparse
import numpy as np
import soundfile as sf
import pyworld as pw
import simpleaudio as sa
import matplotlib.pyplot as plt
from pprint import pprint
import tempfile
import sixel

parser = argparse.ArgumentParser(prog='vocoder.py')
parser.add_argument('-t', '--transpose', type=float, default=6., help='transpose [semitones]')
parser.add_argument('-c', '--correct_pitch', type=float, default=0., help='pitch correction [%%]')
parser.add_argument('-f', '--formant', type=float, default=3., help='formant shift [semitones]')
parser.add_argument('-b', '--breathiness', type=float, default=.3, help='breathiness boost [%%]')
parser.add_argument('-H', '--high_quality', action='store_const', const=True, help='use Harvest instead of Dio')
parser.add_argument('-v', '--visualize', action='store_const', const=True, help='visualize f0, sp and ap with Sixel')
parser.add_argument('in_file', help='input wav file')
parser.add_argument('out_file', nargs='?', help='output wav file (default: direct playback)')

def lerp(a, b, t):
    return a + (b - a) * t

def correct_pitch(f0, rate):
    f0s = np.log2(f0 / 440. + 1e-30) * 12.
    f0sr = np.around(f0s)
    result_s = lerp(f0s, f0sr, rate)
    result = np.power(2., result_s / 12.) * 440.
    return np.where(f0 != 0., result, 0.)

def shift_formant(sp, semitones):
    sp = sp.T
    k = pow(2., semitones/12.)
    sp2 = np.zeros_like(sp)
    n = sp.shape[0]
    for i in range(0, n):
        t = i / (n-1)
        t = min(t / k, 1.)
        c = t * (n-1)
        l = int(c)
        f = c-l
        r = l+1
        if r < n:
            sp2[i] = lerp(sp[l], sp[r], f)
        else:
            sp2[i] = sp[l]
    return sp2.T

def retouch_noise(ap, width, level):
    ap = ap.T
    n = ap.shape[0]
    for i in range(0, n):
        t = i / (n-1) * width * 100.
        t = min(1., t)
        ap[i] = pow(ap[i], lerp(1., 1.-level, t))
    return ap.T

def show_figure(f, log=True):
    epsilon = 1e-8
    plt.figure()
    if len(f.shape) == 1:
        plt.plot(f)
        plt.xlim([0, len(f)])
    elif len(f.shape) == 2:
        if log:
            x = np.log(f + epsilon)
        else:
            x = f + epsilon
        plt.imshow(x.T, origin='lower', interpolation='none', aspect='auto', extent=(0, x.shape[0], 0, x.shape[1]))
    with tempfile.NamedTemporaryFile(prefix='sixel-') as fd:
        plt.savefig(fd, format='png')
        fd.flush()
        writer = sixel.SixelWriter()
        writer.draw(fd.name)

def main(args):
    x, fs = sf.read(args.in_file)
    f0 = None
    t = None

    if args.high_quality:
        f0, t = pw.harvest(x, fs)
    else:
        f0, t = pw.dio(x, fs,
            f0_floor=50.0,
            f0_ceil=600.0,
            channels_in_octave=2,
            frame_period=pw.default_frame_period,
            speed=1)

    f0 = pw.stonemask(x, f0, t, fs)
    sp = pw.cheaptrick(x, f0, t, fs)
    ap = pw.d4c(x, f0, t, fs)

    if args.transpose != 0.:
        f0 *= pow(2., args.transpose/12.)
    if args.correct_pitch != 0.:
        f0 = correct_pitch(f0, args.correct_pitch/100.)
    if args.formant != 0.:
        sp = shift_formant(sp, args.formant)
    if args.breathiness != 0.:
        ap = retouch_noise(ap, .8, args.breathiness/100.)
    y = pw.synthesize(f0, sp, ap, fs, pw.default_frame_period)
    if args.visualize:
        show_figure(f0)
        show_figure(sp)
        show_figure(ap)
    if args.out_file is not None:
        sf.write(args.out_file, y, fs)
    else:
        signal = (y*32767).astype(np.int16)
        pb = sa.play_buffer(signal, 1, 2, fs)
        pb.wait_done()

if __name__ == '__main__':
    args = parser.parse_args()
    main(args)
