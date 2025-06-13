from gtts import gTTS
import os

tts = gTTS(text="iza jest kupkom", lang='pl')
tts.save('file.mp3')

