import torch
from bark import generate_audio, SAMPLE_RATE
import numpy as np
import scipy.io.wavfile as wav
import sys
import os 
import re

sys.stdout.reconfigure(encoding='utf-8')
os.environ["PYTHONIOENCODING"] = "utf-8"
if torch.cuda.is_available():
    torch.set_default_tensor_type("torch.cuda.FloatTensor")
    print("✅ Using GPU (CUDA)")
else:
    print("❌ GPU not available, using CPU")

SPEAKER = "v2/en_speaker_6"

def split_text(text, max_length=150):
    """Splits text into chunks at sentence boundaries (periods, exclamation marks, etc.)."""
    text = text.strip().replace("\n", " ")  # Clean up text
    sentences = re.split(r'(?<=[.!?]) +', text)  # Split at sentence endings

    chunks = []
    current_chunk = ""

    for sentence in sentences:
        if len(current_chunk) + len(sentence) < max_length:
            current_chunk += " " + sentence
        else:
            chunks.append(current_chunk.strip())  # Add current chunk to list
            current_chunk = sentence  # Start a new chunk

    if current_chunk:
        chunks.append(current_chunk.strip())  # Add last chunk

    return chunks

def text_to_speech(text, output_file):
    """Generates speech from text while keeping a consistent voice."""
    text_chunks = split_text(text)
    print(f"🔹 Splitting into {len(text_chunks)} natural chunks")

    audio_arrays = []
    for idx, chunk in enumerate(text_chunks):
        print(f"🔹 Processing chunk {idx+1}/{len(text_chunks)}: {chunk[:50]}...")
        audio = generate_audio(chunk, history_prompt=SPEAKER)
        audio_arrays.append(audio)

    final_audio = np.concatenate(audio_arrays, axis=0)
    wav.write(output_file, rate=SAMPLE_RATE, data=(final_audio * 32767).astype(np.int16))
    print(f"✅ Voiceover saved to {output_file}")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("❌ Error: No text input provided!")
        sys.exit(1)

    text_to_speech(sys.argv[1], "voiceover.wav")