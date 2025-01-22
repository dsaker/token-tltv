const fromLangSelect = document.getElementById("from-lang-select");
const fromVoiceDiv = document.getElementById("from-voice-div");
const fromVoiceOptions = document.getElementsByName("from-voice-option");
const toLangSelect = document.getElementById("to-lang-select");
const toVoiceDiv = document.getElementById("to-voice-div");
const toVoiceOptions = document.getElementsByName("to-voice-option");
const translateForm = document.getElementById("translate-form");
const ldsDiv = document.getElementById("lds-div");
const tokenInput = document.getElementById("token-input");
const titleInput = document.getElementById("title-input");
const errorDiv = document.getElementById("error-div");
const submitForm = document.getElementById("submit-form");

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

submitForm.addEventListener("click", () => {
	translateForm.style.display = "none";
	ldsDiv.style.display = "block";
});