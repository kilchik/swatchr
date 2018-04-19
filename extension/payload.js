// send the page title as a chrome message
chrome.runtime.sendMessage({title: document.title, content: document.body.innerHTML});
