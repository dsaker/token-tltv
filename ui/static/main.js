const fromLangSelect = document.getElementById("from-lang-select");
const fromVoiceDiv = document.getElementById("from-voice-div");
const fromVoiceOptions = document.getElementsByName("from-voice-option");
const toLangSelect = document.getElementById("to-lang-select");
const toVoiceDiv = document.getElementById("to-voice-div");
const toVoiceOptions = document.getElementsByName("to-voice-option");
const audioForm = document.getElementById("audio-form");
const ldsDiv = document.getElementById("lds-div");
const divFlash = document.getElementById("div-flash")
const parseForm = document.getElementById("parse-form");

fromLangSelect.addEventListener("change", () => {
	let langId = fromLangSelect.value;
	fromVoiceOptions.forEach((elem) => {
		if (elem.classList.contains(langId)) {
			elem.style.display = "block";
		}
	})
	fromVoiceDiv.style.display = "block";
})

toLangSelect.addEventListener("change", () => {
	let langId = toLangSelect.value;
	toVoiceOptions.forEach((elem) => {
		if (elem.classList.contains(langId)) {
			elem.style.display = "block";
		}
	})
	toVoiceDiv.style.display = "block";
})

let filename = ""
function sendData(url) {
	divFlash.style.display = "none";
	divFlash.innerHTML = "";
	// Associate the FormData object with the form element
	const formData = new FormData(audioForm);
	fetch(url, {
		method: "POST",
		// Set the FormData instance as the request body
		body: formData,
	}).then(async (response) => {
		if (!response.ok) {
			throw Error(await response.text());
		}
		// We are reading the *Content-Disposition* header for getting the original filename given from the server
		const header = response.headers.get('Content-Disposition');
		const parts = header.split(';');
		filename = parts[1].split('=')[1].replaceAll("\"", "");
		return response.blob()
	}).then((blob) => {
			if (blob != null) {
				let url = window.URL.createObjectURL(blob);
				let a = document.createElement('a');
				a.href = url;
				a.download = filename;
				document.body.appendChild(a);
				a.click();
				a.remove();
			}
			audioForm.style.display = "none";
			ldsDiv.style.display = "block";
		})
		.catch((message) => {
			divFlash.style.display = "block";
			divFlash.innerHTML = message;
		});

}

// Take over form submission
audioForm.addEventListener("submit", (event) => {
	event.preventDefault();
	sendData("/v1/audio")
});

// Take over form submission
parseForm.addEventListener("submit", (event) => {
	event.preventDefault();
	sendData("/v1/parse")
});