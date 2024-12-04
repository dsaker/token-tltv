import json

"""voices_file.py

This script loads a file from local and prints the voices in struct form in
/internal/models/models.go
"""


def print_voices(results):
    global language
    with open('google_voices.json', 'w', encoding='utf-8') as f:
        json.dump(results, f, ensure_ascii=False, indent=4)

    # Open the JSON file
    with open('google_languages.json', 'r') as f1:
        # Load the JSON data into a Python dictionary
        languages = json.load(f1)
    voices = results['voices']
    for i, voice in enumerate(voices):
        lang_code = voice["languageCodes"][0]
        # get the language id for the voice from the language tag
        lang_tag = lang_code.split("-")
        found = False
        lang_id = -1
        for j, lang in enumerate(languages):
            # filipino voice lang_tag does not match language tag
            if lang_tag[0] == 'fil' and lang['language'] == 'tl':
                found = True
                lang_id = j
            # norwegian voice lang_tag does not match language tag
            if lang_tag[0] == 'nb' and lang['language'] == 'no':
                found = True
                lang_id = j
            if lang['language'] == lang_tag[0]:
                found = True
                lang_id = j
                break
        if not found:
            print("lang_id not found for " + voice['name'] + voice['languageCodes'][0])
        else:
            # build string that corresponds to voices struct in /internal/models/models.go
            print(str(i) + ":  {LanguageID: " + str(lang_id) + ", LanguageCodes: {", end='')
            build_string = ""
            for _, l in enumerate(voice["languageCodes"]):
                build_string += "\"" + l + "\"" + ","
            build_string = build_string[:-1]
            print(build_string + "}, SsmlGender: \"" + voice['ssmlGender'] + "\", Name: \"" + voice[
                      'name'] + "\", NaturalSampleRateHertz: " + str(voice['naturalSampleRateHertz']) + "},")


if __name__ == "__main__":
    # Load JSON data
    with open('google_voices.json', 'r') as file:
        data = json.load(file)
    print_voices(data)
