chrome.extension.onMessage.addListener(
    function(request, sender, sendResponse) {
        console.log(request.content.match(/(magnet:.+)\" class.*title=\"Скачать по magnet-ссылке\">/)[1]);
    });

chrome.tabs.onUpdated.addListener(function(tab) {
    chrome.tabs.executeScript(tab.id, {
        code: "chrome.extension.sendMessage({content: document.body.innerHTML}, function(response) { console.log('success'); });"
    }, function() { console.log('done'); });

});
