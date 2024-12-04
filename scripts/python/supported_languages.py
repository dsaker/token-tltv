import json
from google.cloud import translate_v2 as translate


"""supported_languages.py

supported_languages.py makes a request to google translate and requests all of the 
supported languages in the cloud translation api and prints them in the Language struct
form in /internal/models/models.go 

for more info -> https://cloud.google.com/translate/docs/languages
"""


def list_languages():
    """Lists all available languages."""

    translate_client = translate.Client()

    results = translate_client.get_languages()

    with open('google_languages.json', 'w', encoding='utf-8') as f:
        json.dump(results, f, ensure_ascii=False, indent=4)

    for i, lang in enumerate(results):
        print(str(i) + ": {Name: \"" + lang['name'] + "\", Tag: \"" + lang['language'] + "\"},")


if __name__ == "__main__":
    list_languages()
