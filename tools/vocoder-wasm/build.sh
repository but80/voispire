#!/bin/bash

emcc vocoder.cpp ../../cmodules/world/src/*.cpp \
  -I../../cmodules/world/src -lstdc++ -lm \
  -o vocoder.html \
  -s EXPORTED_FUNCTIONS='["_vocoder"]' \
  -s EXTRA_EXPORTED_RUNTIME_METHODS='["ccall", "cwrap"]' \
  -s ALLOW_MEMORY_GROWTH=1 \
  --shell-file template.html
