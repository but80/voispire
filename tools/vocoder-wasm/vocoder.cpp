#include <stdio.h>
#include <math.h>
#include <emscripten/emscripten.h>
#include "world/dio.h"
#include "world/harvest.h"
#include "world/stonemask.h"
#include "world/cheaptrick.h"
#include "world/d4c.h"
#include "world/synthesis.h"

double lerp(double a, double b, double t) {
	return a + (b - a) * t;
}

void shiftFormant(double** spectro, int f0Length, int fftSizeHalf, double semitones, double** outSpectro) {
	double rate = pow(2.0, semitones/12.0);
	for (int i=0; i<fftSizeHalf; i++) {
		double t = double(i) / double(fftSizeHalf-1) / rate;
		if (1.0 < t) t = 1.0;
		double c = t * double(fftSizeHalf-1);
		int l = int(c);
		int r = l+1;
		double f = c - double(l);
		for (int j=0; j<f0Length; j++) {
			outSpectro[j][i] = r < fftSizeHalf ?
				lerp(spectro[j][l], spectro[j][r], f) :
				spectro[j][l];
		}
	}
}

void retouchNoise(double** aperiod, int f0Length, int fftSizeHalf, double width, double level) {
	for (int i=0; i<fftSizeHalf; i++) {
		double t = double(i) / double(fftSizeHalf-1) * width * 100.0;
		if (1.0 < t) t = 1.0;
		for (int j=0; j<f0Length; j++) {
			aperiod[j][i] = pow(aperiod[j][i], lerp(1.0, 1.0-level, t));
		}
	}
}

extern "C" {

EMSCRIPTEN_KEEPALIVE
void vocoder(double* x, int xLength, int fs, double framePeriodMsec, double f0Floor, double f0Ceil, bool useHarvest, double confF0Shift, double confFormantShift, double confBreathiness) {
	int f0Length;
	double* tmppos;
	double* f0h;

	// Estimate f0
	printf("estimating f0\n");
	if (useHarvest) {
		HarvestOption opt;
		InitializeHarvestOption(&opt);
		opt.f0_floor = f0Floor;
		opt.f0_ceil = f0Ceil;
		opt.frame_period = framePeriodMsec;
		f0Length = GetSamplesForHarvest(fs, xLength, framePeriodMsec);
		tmppos = new double[f0Length];
		f0h = new double[f0Length];
		Harvest(x, xLength, fs, &opt, tmppos, f0h);
	} else {
		DioOption opt;
		InitializeDioOption(&opt);
		opt.f0_floor = f0Floor;
		opt.f0_ceil = f0Ceil;
		opt.frame_period = framePeriodMsec;
		f0Length = GetSamplesForDIO(fs, xLength, framePeriodMsec);
		tmppos = new double[f0Length];
		f0h = new double[f0Length];
		Dio(x, xLength, fs, &opt, tmppos, f0h);
	}

	// Refine f0
	printf("refining f0\n");
	double* f0 = new double[f0Length];
	StoneMask(x, xLength, fs, tmppos, f0h, f0Length, f0);
	delete[] f0h;

	// Estimate spectrogram
	printf("estimating spectrogram\n");
	CheapTrickOption cheapTrickOption;
	InitializeCheapTrickOption(fs, &cheapTrickOption);
	int fftSize = cheapTrickOption.fft_size;
	int fftSizeHalf = fftSize / 2 + 1;
	double* spectroArray = new double[f0Length * fftSizeHalf];
	double** spectro = new double*[f0Length];
	for (int i=0; i<f0Length; i++) {
		spectro[i] = &spectroArray[i * fftSizeHalf];
	}
	CheapTrick(x, xLength, fs, tmppos, f0, f0Length, &cheapTrickOption, spectro);

	// Estimate aperiodicity
	printf("estimating aperiodicity\n");
	D4COption d4cOption;
	InitializeD4COption(&d4cOption);
	double* aperiodArray = new double[f0Length * fftSizeHalf];
	double** aperiod = new double*[f0Length];
	for (int i=0; i<f0Length; i++) {
		aperiod[i] = &aperiodArray[i * fftSizeHalf];
	}
	D4C(x, xLength, fs, tmppos, f0, f0Length, fftSize, &d4cOption, aperiod);

	// Retouch
	printf("retouching\n");

	double f0ShiftRate = pow(2.0, confF0Shift/12.0);
	for (int i=0; i<f0Length; i++) f0[i] *= f0ShiftRate;

	double* newSpectroArray = new double[f0Length * fftSizeHalf];
	double** newSpectro = new double*[f0Length];
	for (int i=0; i<f0Length; i++) {
		newSpectro[i] = &newSpectroArray[i * fftSizeHalf];
	}
	shiftFormant(spectro, f0Length, fftSizeHalf, confFormantShift, newSpectro);
	delete[] spectro;
	delete[] spectroArray;

	retouchNoise(aperiod, f0Length, fftSizeHalf, .8, confBreathiness/100.0);

	// Synthesize
	printf("synthesizing\n");
	Synthesis(f0, f0Length, newSpectro, aperiod, fftSize, framePeriodMsec, fs, xLength, x);

	// Finalize
	printf("done\n");
	delete[] aperiod;
	delete[] aperiodArray;
	delete[] newSpectro;
	delete[] newSpectroArray;
	delete[] f0;
	delete[] tmppos;
}

}
