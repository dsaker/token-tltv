import json
import os
import requests

api_key = os.environ['API_KEY']

"""voices_api.py

voices_api.py makes a request to google text-to-speech and downloads all of the 
supported voices in the cloud text-to-speech api. 

for more info -> https://cloud.google.com/text-to-speech/docs/voices
and https://cloud.google.com/text-to-speech/docs/reference/rest/v1/voices/list

you must get an api key to perform this request ->
https://cloud.google.com/docs/authentication/api-keys

must export api key
export API_KEY=<api key>

voices struct at /internal/models/models.go
"""


def print_voices(results):
    global language
    with open('google_voices.json', 'w', encoding='utf-8') as f:
        json.dump(results, f, ensure_ascii=False, indent=4)

    # Open the JSON file
    with open('google_languages.json', 'r') as file:
        # Load the JSON data into a Python dictionary
        data = json.load(file)
    for i, voice in enumerate(results):
        for lang_code in voice["languageCodes"]:
            # get the language id for the voice from the language tag
            lang_tag = lang_code.split("-")
            language = data[lang_tag]
        print("{ID: " + str(i) + ", LanguageId: " + language['ID'] + ", LanguageCodes: " + voice["languageCodes"] + ", SsmlGender: \"" + voice['ssmlGender'] + "\", Name: \"" + voice['name'] + "\", NaturalSampleRateHertz: " + voice['naturalSampleRateHertz'], end='')


def main():
    # Set up the API endpoint
    url = f'https://texttospeech.googleapis.com/v1/voices?key={api_key}'

    # Make the GET request
    response = requests.get(url)

    # Check if the request was successful
    if response.status_code == 200:
        voices = response.json()
        # print(json.dumps(voices, indent=2))
        print_voices(voices)
    else:
        print(f"Failed to retrieve voices. Status code: {response.status_code}")
        print(response.text)


if __name__ == "__main__":
    main()
