from gtts import gTTS
import sys

uster_tts = ""
for a in sys.argv[1:]:
    uster_tts += a + " "

tts = gTTS(text=uster_tts, lang="pl")
tts.save("file.mp3")
