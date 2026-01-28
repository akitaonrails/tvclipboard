/* exported loadTranslations, t, formatString */
// i18n helper functions

let translations = {};

// Load translations from server-side injection
function loadTranslations() {
	if (window.translations) {
		translations = window.translations;
		console.log('Translations loaded:', Object.keys(translations.common).length + ' common, ' + 
				Object.keys(translations.host).length + ' host, ' +
				Object.keys(translations.client).length + ' client, ' +
				Object.keys(translations.errors).length + ' error keys');
	} else {
		console.warn('Translations not available');
	}
}

// Get a translation by key (e.g., 'host.title', 'errors.no_token')
// Supports optional params object for placeholder substitution
function t(key, params, fallback) {
	if (!translations || typeof translations !== 'object') {
		console.warn('Translations not loaded, key:', key);
		return fallback || key;
	}

	const parts = key.split('.');
	let section, name;

	if (parts.length === 2) {
		[section, name] = parts;
	} else {
		// Try all sections if no section specified
		name = key;
		if (translations.common && translations.common[name]) {
			return formatString(translations.common[name], params);
		}
		if (translations.host && translations.host[name]) {
			return formatString(translations.host[name], params);
		}
		if (translations.client && translations.client[name]) {
			return formatString(translations.client[name], params);
		}
		if (translations.errors && translations.errors[name]) {
			return formatString(translations.errors[name], params);
		}
		return fallback || key;
	}

	let sectionObj = translations[section];
	if (!sectionObj) {
		console.warn('Translation section not found:', section, 'for key:', key);
		return fallback || key;
	}

	const result = sectionObj[name];
	if (!result) {
		console.warn('Translation key not found:', name, 'in section:', section);
		return fallback || key;
	}

	return formatString(result, params);
}

// Format string with params (supports {key} placeholders)
function formatString(str, params) {
	if (!params || typeof params !== 'object') {
		return str;
	}
	return str.replace(/\{(\w+)\}/g, (match, key) => {
		return params[key] !== undefined ? params[key] : match;
	});
}

// Apply translations to elements with data-i18n attribute
function applyTranslations() {
	loadTranslations();

	const elements = document.querySelectorAll('[data-i18n]');
	elements.forEach(element => {
		const key = element.getAttribute('data-i18n');
		if (key) {
			const translation = t(key);
			const before = element.getAttribute('data-i18n-before') || '';
			if (element.tagName === 'INPUT' || element.tagName === 'TEXTAREA') {
				element.placeholder = before + translation;
			} else {
				element.textContent = before + translation;
			}
		}
	});

	const titleElements = document.querySelectorAll('[data-i18n-title]');
	titleElements.forEach(element => {
		const key = element.getAttribute('data-i18n-title');
		if (key) {
			element.title = t(key);
		}
	});

	const placeholderElements = document.querySelectorAll('[data-i18n-placeholder]');
	placeholderElements.forEach(element => {
		const key = element.getAttribute('data-i18n-placeholder');
		if (key) {
			element.placeholder = t(key);
		}
	});
}

// Load translations on page load
document.addEventListener('DOMContentLoaded', applyTranslations);
