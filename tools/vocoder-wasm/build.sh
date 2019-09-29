#!/bin/bash

emcc vocoder.cpp ../../cmodules/world/src/*.cpp \
  -I../../cmodules/world/src -lstdc++ -lm \
  -o vocoder.html \
  --shell-file template.html \
  -s WASM=1 \
  -s ALLOW_MEMORY_GROWTH=1 \
  -s EXTRA_EXPORTED_RUNTIME_METHODS='["ccall", "cwrap"]' \
  -s EXPORTED_FUNCTIONS='["_vocoder"]'
