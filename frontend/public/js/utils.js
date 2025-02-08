// Tab functionality
function openTab(evt, tabName) {
    const tabcontent = document.getElementsByClassName("tabcontent");
    for (let i = 0; i < tabcontent.length; i++) {
        tabcontent[i].style.display = "none";
    }

    const tablinks = document.getElementsByClassName("tablinks");
    for (let i = 0; i < tablinks.length; i++) {
        tablinks[i].className = tablinks[i].className.replace(" active", "");
    }

    document.getElementById(tabName).style.display = "block";
    evt.currentTarget.className += " active";
}

// Display result in textarea
function displayResult(id, result) {
    document.getElementById(id).value = JSON.stringify(result, null, 2);
}

// Handle and display error in textarea
function handleError(id, err) {
    document.getElementById(id).value = "Error: " + err.message;
}

// Extract base filename from path
function filepathBaseName(path) {
    return path.split('/').pop();
}

// Initialize the application
function initApp() {
    // Set default tab
    document.getElementById("defaultOpen").click();
    
    // Initialize video functionality
    initVideo();
    
    // Add any other initialization here
}
