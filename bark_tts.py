import torch
from bark import generate_audio
import numpy as np
import scipy.io.wavfile as wav
import sys

if torch.cuda.is_available():
    torch.set_default_tensor_type("torch.cuda.FloatTensor")
    print("✅ Using GPU (CUDA)")
else:
    print("❌ GPU not available, using CPU")

def text_to_speech(text, output_file):
    audio_array = generate_audio(text)  # 🔹 No need to pass `device`
    wav.write(output_file, rate=24000, data=(audio_array * 32767).astype(np.int16))
    print(f"✅ Voiceover saved to {output_file}")

if __name__ == "__main__":
    print("✅ Python script started successfully!")
    if len(sys.argv) < 2:
        print("❌ Error: No text input provided!")
        sys.exit(1)

    print(f"🔹 Received text: {sys.argv[1]}")
    text_to_speech(sys.argv[1], "voiceover.wav")
