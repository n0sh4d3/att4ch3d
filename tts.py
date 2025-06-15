from llama_cpp import Llama
from gtts import gTTS
import sys
import argostranslate.package
import argostranslate.translate

packages = argostranslate.package.get_available_packages()


# fuck this shit
user_input = " ".join(sys.argv[1:])

pl_en_package = next(p for p in packages if p.from_code == "pl" and p.to_code == "en")
argostranslate.package.install_from_path(pl_en_package.download())

user_input = argostranslate.translate.translate(user_input, "pl", "en")

llm = Llama(
    model_path="./model.gguf",
    n_gpu_layers=-1,  # GPU GPU!!!
    verbose=False,  # stfu performance output
)

output = llm(
    f"Q: {user_input} A: ",
    max_tokens=64,
    stop=["Q:"],
    echo=True,
)

full_text = output["choices"][0]["text"]
if "A: " in full_text:
    response_text = full_text.split("A: ", 1)[1].strip()
else:
    response_text = full_text.strip()

pl_en_package = next(p for p in packages if p.from_code == "en" and p.to_code == "pl")
argostranslate.package.install_from_path(pl_en_package.download())

translated = argostranslate.translate.translate(response_text, "en", "pl")
print(translated)

tts = gTTS(text=translated, lang="pl")
tts.save("file.mp3")

with open("debug.txt", "w") as f:
    f.write(translated)

print("zapisano szef")
