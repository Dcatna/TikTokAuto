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
    print("using cuda")
else:
    print("using cpu")

SPEAKER = "v2/en_speaker_1"

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
    text = text.strip()  # Remove extra spaces/newlines
    if not text:
        print("❌ Error: Text is empty, skipping speech generation")
        return

    text_chunks = split_text(text)
    if not text_chunks:
        print("❌ Error: No valid text chunks found")
        return

    audio_arrays = []
    for idx, chunk in enumerate(text_chunks):
        chunk = chunk.strip()
        if not chunk:
            print(f"⚠️ Skipping empty chunk at index {idx}")
            continue  # Skip empty text chunks

        print(f"🔹 Generating audio for chunk {idx+1}/{len(text_chunks)}: {chunk[:50]}...")
        try:
            audio = generate_audio(chunk, history_prompt=SPEAKER)
            audio_arrays.append(audio)
        except Exception as e:
            print(f"❌ Error generating audio for chunk {idx+1}: {e}")
            continue  # Continue processing remaining chunks

    if audio_arrays:
        final_audio = np.concatenate(audio_arrays, axis=0)
        wav.write(output_file, rate=SAMPLE_RATE, data=(final_audio * 32767).astype(np.int16))
        print(f"✅ Voiceover saved to {output_file}")
    else:
        print("❌ Error: No valid audio generated")


if __name__ == "__main__":

    text_to_speech(sys.argv[1], "voiceover.wav")