let uploadStatusCheckInterval = null;

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function updateProgressBar(percent) {
    const progressContainer = document.getElementById('uploadProgress');
    const progressBar = document.getElementById('uploadProgressBar');
    progressContainer.style.display = 'block';
    progressBar.style.width = `${percent}%`;
    progressBar.textContent = `${Math.round(percent)}%`;
}

function updateUploadStatus(fileId) {
    fetch(`/video/status/${fileId}`)
        .then(res => res.json())
        .then(data => {
            if (data.data) {
                const status = data.data;
                const statusContainer = document.getElementById('uploadStatus');
                const progressContainer = document.getElementById('uploadProgress');
                
                // Only show status container for completed or failed states
                if (['completed', 'failed'].includes(status.status)) {
                    // Stop polling
                    clearInterval(uploadStatusCheckInterval);
                    uploadStatusCheckInterval = null;

                    // Hide progress bar
                    progressContainer.style.display = 'none';

                    // Show final status
                    statusContainer.style.display = 'block';
                    statusContainer.className = `status-container status-${status.status}`;

                    document.getElementById('statusText').textContent = status.status;
                    document.getElementById('statusMessage').textContent = status.message || '-';
                    document.getElementById('statusFileSize').textContent = formatFileSize(status.fileSize);
                    document.getElementById('statusIpfsCid').textContent = status.ipfsCid || '-';
                    document.getElementById('statusUpdatedAt').textContent = new Date(status.updatedAt).toLocaleString();
                } else {
                    statusContainer.style.display = 'none';
                }

                // Extract progress percentage from status message if available
                let progress = 0;
                if (status.message) {
                    const match = status.message.match(/(\d+(\.\d+)?)%/);
                    if (match) {
                        progress = parseFloat(match[1]);
                    }
                }

                // Update progress bar based on status
                if (status.status === 'uploading') {
                    updateProgressBar(progress);
                } else if (status.status === 'completed') {
                    updateProgressBar(100);
                }
            }
        })
        .catch(err => {
            console.error('Error checking upload status:', err);
            clearInterval(uploadStatusCheckInterval);
            uploadStatusCheckInterval = null;
            handleError("uploadResult", err);
        });
}

// Initialize video upload form handler
function initVideoUpload() {
    const MAX_FILE_SIZE = 100 * 1024 * 1024; // 100 MB in bytes
    const form = document.getElementById("uploadForm");
    if (!form) {
        console.error('Upload form not found');
        return;
    }

    // Add file input change handler for instant feedback
    const fileInput = document.getElementById("videoFile");
    fileInput.addEventListener('change', function(e) {
        const file = e.target.files[0];
        if (file) {
            if (file.size > MAX_FILE_SIZE) {
                alert(`File size (${formatFileSize(file.size)}) exceeds maximum allowed size of ${formatFileSize(MAX_FILE_SIZE)}`);
                e.target.value = ''; // Clear the file input
            }
        }
    });

    form.addEventListener("submit", function (e) {
        e.preventDefault();
        console.log('Upload form submitted');

        const file = fileInput.files[0];
        if (!file) {
            alert("Please select a video file.");
            return;
        }

        // Double-check file size before upload
        if (file.size > MAX_FILE_SIZE) {
            alert(`File size (${formatFileSize(file.size)}) exceeds maximum allowed size of ${formatFileSize(MAX_FILE_SIZE)}`);
            return;
        }

        const title = document.getElementById("title").value.trim();
        const description = document.getElementById("description").value.trim();
        const formData = new FormData();
        formData.append("video", file);
        formData.append("title", title);
        formData.append("description", description);

        // Reset and show progress bar
        const progressContainer = document.getElementById('uploadProgress');
        const progressBar = document.getElementById('uploadProgressBar');
        if (!progressContainer || !progressBar) {
            console.error('Progress elements not found');
            return;
        }

        progressContainer.style.display = 'block';
        updateProgressBar(0);

        // Reset and hide status container
        const statusContainer = document.getElementById('uploadStatus');
        if (statusContainer) {
            statusContainer.style.display = 'none';
        }

        // Clear any existing interval
        if (uploadStatusCheckInterval) {
            clearInterval(uploadStatusCheckInterval);
        }

        // Create XMLHttpRequest to track initial upload progress
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/video/upload', true);

        // Add upload progress handler
        xhr.upload.onprogress = function(e) {
            if (e.lengthComputable) {
                const percent = (e.loaded / e.total) * 100;
                console.log(`Upload progress: ${percent.toFixed(2)}%`);
                updateProgressBar(percent);
            }
        };

        // Add upload start handler
        xhr.upload.onloadstart = function() {
            console.log('Upload started');
            progressContainer.style.display = 'block';
            updateProgressBar(0);
        };

        // Add upload complete handler
        xhr.upload.onload = function() {
            console.log('Upload completed');
            updateProgressBar(100);
        };

        // Add upload error handler
        xhr.upload.onerror = function() {
            console.error('Upload failed');
            progressContainer.style.display = 'none';
            if (statusContainer) {
                statusContainer.style.display = 'block';
                statusContainer.className = 'status-container status-failed';
                document.getElementById('statusText').textContent = 'failed';
                document.getElementById('statusMessage').textContent = 'Upload failed due to network error';
            }
        };

        xhr.onload = function () {
            console.log('Upload completed, status:', xhr.status);
            if (xhr.status === 200) {
                try {
                    const response = JSON.parse(xhr.responseText);
                    console.log('Upload response:', response);
                    displayResult("uploadResult", response);
                    if (response.data && response.data.video && response.data.video.fileId) {
                        // Start checking upload status immediately
                        updateUploadStatus(response.data.video.fileId);
                        // Set interval to check status every 1 second
                        uploadStatusCheckInterval = setInterval(() => {
                            updateUploadStatus(response.data.video.fileId);
                        }, 1000);
                    }
                } catch (err) {
                    console.error('Error parsing response:', err);
                    handleError("uploadResult", new Error('Invalid response format'));
                }
            } else {
                console.error('Upload failed:', xhr.statusText);
                try {
                    const errorResponse = JSON.parse(xhr.responseText);
                    if (errorResponse.error) {
                        handleError("uploadResult", new Error(errorResponse.error.message || errorResponse.error));
                        progressContainer.style.display = 'none';
                        if (statusContainer) {
                            statusContainer.style.display = 'block';
                            statusContainer.className = 'status-container status-failed';
                            document.getElementById('statusText').textContent = 'failed';
                            document.getElementById('statusMessage').textContent = errorResponse.error.message || errorResponse.error;
                        }
                    } else {
                        handleError("uploadResult", new Error(`Upload failed: ${xhr.statusText}`));
                    }
                } catch (err) {
                    handleError("uploadResult", new Error(`Upload failed: ${xhr.statusText}`));
                }
            }
        };

        console.log('Starting upload...');
        xhr.send(formData);
    });
}

function transcodeVideo() {
    console.log('transcodeVideo function called');
    const transcodeOption = document.querySelector('input[name="transcodeOption"]:checked').value;
    console.log('Selected transcode option:', transcodeOption);
    
    if (transcodeOption === "cid") {
        const cid = document.getElementById("transcodeCIDInput").value.trim();
        console.log('Input CID value:', cid);
        
        if (!cid) {
            console.log('No CID provided, showing alert');
            alert("Please enter a CID for transcoding.");
            return;
        }
        
        const payload = { cid: cid };
        console.log('Preparing transcode request with payload:', payload);
        const jsonBody = JSON.stringify(payload);
        console.log('Request JSON body:', jsonBody);
        
        fetch("/video/transcode", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Accept": "application/json"
            },
            body: jsonBody
        })
        .then(response => {
            console.log('Response status:', response.status);
            console.log('Response headers:', response.headers);
            return response.text().then(text => {
                try {
                    console.log('Raw response text:', text);
                    const data = JSON.parse(text);
                    console.log('Parsed response data:', data);
                    if (!response.ok) {
                        throw data;
                    }
                    return data;
                } catch (e) {
                    console.error('JSON parse error:', e);
                    throw new Error('Invalid JSON response: ' + text);
                }
            });
        })
        .then(data => {
            console.log('Successfully processed response data:', data);
            displayResult("transcodeResult", data);
        })
        .catch(error => {
            console.error('Transcode error:', error);
            console.error('Error details:', error);
            handleError("transcodeResult", error);
        });
    } else {
        console.log('Invalid transcode option, showing alert');
        alert("Please upload your video first using the upload form, then use the returned CID for transcoding.");
    }
}

function listVideos() {
    fetch("/video/list")
        .then(res => res.json())
        .then(data => {
            displayResult("listResult", data);
            if (data.data && Array.isArray(data.data)) {
                displayTranscodedVideos(data.data);
            }
        })
        .catch(err => handleError("listResult", err));
}

function displayTranscodedVideos(videos) {
    const container = document.getElementById('videoList');
    container.innerHTML = '';

    if (videos.length === 0) {
        container.innerHTML = '<p>No videos available.</p>';
        return;
    }

    videos.forEach(video => {
        const videoElement = document.createElement('div');
        videoElement.className = 'video-item';

        const title = document.createElement('h3');
        title.textContent = video.title;

        const description = document.createElement('p');
        description.textContent = video.description || 'No description available';

        const details = document.createElement('div');
        details.className = 'video-details';
        
        // Create status badge
        const statusBadge = document.createElement('span');
        statusBadge.className = `status-badge ${video.uploadStatus}`;
        statusBadge.textContent = video.uploadStatus;
        
        details.innerHTML = `
            <p><strong>Status:</strong> ${statusBadge.outerHTML}</p>
            <p><strong>File Size:</strong> ${formatFileSize(video.fileSize)}</p>
            <p><strong>Created:</strong> ${new Date(video.createdAt).toLocaleString()}</p>
            <p><strong>Updated:</strong> ${new Date(video.updatedAt).toLocaleString()}</p>
            ${video.ipfsCid ? `<p><strong>IPFS CID:</strong> ${video.ipfsCid}</p>` : ''}
        `;

        // Set the status attribute for styling
        videoElement.setAttribute('data-status', video.uploadStatus);

        const links = document.createElement('div');
        links.className = 'link-list';

        if (video.ipfsCid) {
            // Add links for different video qualities if available
            if (video.variants) {
                Object.entries(video.variants).forEach(([quality, cid]) => {
                    const link = document.createElement('a');
                    link.href = `/ipfs/${cid}`;
                    link.textContent = `View ${quality}`;
                    link.target = '_blank';
                    links.appendChild(link);
                });
            }

            // Add link to original file
            const originalLink = document.createElement('a');
            originalLink.href = `/ipfs/${video.ipfsCid}`;
            originalLink.textContent = 'View Original';
            originalLink.target = '_blank';
            links.appendChild(originalLink);
        }

        videoElement.appendChild(title);
        videoElement.appendChild(description);
        videoElement.appendChild(details);
        videoElement.appendChild(links);

        container.appendChild(videoElement);
    });
}

// Initialize all video-related functionality
function initVideo() {
    initVideoUpload();

    // Add event listener for transcode option radio buttons
    document.querySelectorAll('input[name="transcodeOption"]').forEach(radio => {
        radio.addEventListener('change', function () {
            document.getElementById('transcodeCIDForm').style.display =
                this.value === 'cid' ? 'block' : 'none';
            document.getElementById('transcodeUploadForm').style.display =
                this.value === 'upload' ? 'block' : 'none';
        });
    });

    // Initial list of videos
    listVideos();
}
