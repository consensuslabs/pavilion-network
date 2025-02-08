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
                statusContainer.style.display = 'block';
                statusContainer.className = `status-container status-${status.status}`;

                document.getElementById('statusText').textContent = status.status;
                document.getElementById('statusMessage').textContent = status.message || '-';
                document.getElementById('statusFileSize').textContent = formatFileSize(status.fileSize);
                document.getElementById('statusChecksum').textContent = status.checksum || '-';
                document.getElementById('statusIpfsCid').textContent = status.ipfsCid || '-';
                document.getElementById('statusUpdatedAt').textContent = new Date(status.updatedAt).toLocaleString();

                // Update progress bar based on status
                if (status.status === 'uploading' && typeof status.progress === 'number') {
                    updateProgressBar(status.progress);
                } else if (status.status === 'completed') {
                    updateProgressBar(100);
                }

                // Stop checking if we reach a final state
                if (['completed', 'failed'].includes(status.status)) {
                    clearInterval(uploadStatusCheckInterval);
                }
            }
        })
        .catch(err => {
            console.error('Error checking upload status:', err);
            clearInterval(uploadStatusCheckInterval);
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

        // Reset status container
        const statusContainer = document.getElementById('uploadStatus');
        if (statusContainer) {
            statusContainer.style.display = 'none';
        }

        // Clear any existing interval
        if (uploadStatusCheckInterval) {
            clearInterval(uploadStatusCheckInterval);
        }

        // Create XMLHttpRequest to track upload progress
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/video/upload', true);

        xhr.upload.onprogress = function (e) {
            if (e.lengthComputable) {
                const percentComplete = (e.loaded / e.total) * 100;
                console.log(`Upload progress: ${percentComplete}%`);
                updateProgressBar(percentComplete);
            }
        };

        xhr.onload = function () {
            console.log('Upload completed, status:', xhr.status);
            if (xhr.status === 200) {
                try {
                    const response = JSON.parse(xhr.responseText);
                    console.log('Upload response:', response);
                    displayResult("uploadResult", response);
                    if (response.data && response.data.fileId) {
                        updateUploadStatus(response.data.fileId);
                        uploadStatusCheckInterval = setInterval(() => {
                            updateUploadStatus(response.data.fileId);
                        }, 2000);
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
                        handleError("uploadResult", new Error(errorResponse.error));
                        // Hide progress bar on error
                        progressContainer.style.display = 'none';
                        // Show error status
                        if (statusContainer) {
                            statusContainer.style.display = 'block';
                            statusContainer.className = 'status-container status-failed';
                            document.getElementById('statusText').textContent = 'Failed';
                            document.getElementById('statusMessage').textContent = errorResponse.error;
                        }
                    } else {
                        handleError("uploadResult", new Error(`Upload failed: ${xhr.statusText}`));
                    }
                } catch (err) {
                    handleError("uploadResult", new Error(`Upload failed: ${xhr.statusText}`));
                }
            }
        };

        xhr.onerror = function () {
            console.error('Upload network error');
            handleError("uploadResult", new Error('Upload failed: Network error'));
            // Hide progress bar on error
            progressContainer.style.display = 'none';
            // Show error status
            if (statusContainer) {
                statusContainer.style.display = 'block';
                statusContainer.className = 'status-container status-failed';
                document.getElementById('statusText').textContent = 'Failed';
                document.getElementById('statusMessage').textContent = 'Network error occurred during upload';
            }
        };

        console.log('Starting upload...');
        xhr.send(formData);
    });
}

function transcodeVideo() {
    const transcodeOption = document.querySelector('input[name="transcodeOption"]:checked').value;
    let payload = {};
    if (transcodeOption === "cid") {
        const sourceCID = document.getElementById("transcodeCIDInput").value.trim();
        if (!sourceCID) {
            alert("Please enter a CID for transcoding.");
            return;
        }
        payload = {
            sourceCID: sourceCID
        };
    } else {
        const file = document.getElementById("transcodeFile").files[0];
        if (!file) {
            alert("Please select a video file for transcoding.");
            return;
        }
        const title = document.getElementById("transcodeTitle").value.trim();
        const description = document.getElementById("transcodeDescription").value.trim();

        const formData = new FormData();
        formData.append("video", file);
        formData.append("title", title);
        formData.append("description", description);

        payload = formData;
    }

    transcodeWithPayload(payload);
}

function transcodeWithPayload(payload) {
    fetch("/video/transcode", {
        method: "POST",
        body: payload
    })
        .then(res => res.json())
        .then(data => displayResult("transcodeResult", data))
        .catch(err => handleError("transcodeResult", err));
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
        details.innerHTML = `
            <p><strong>Status:</strong> ${video.status}</p>
            <p><strong>File Size:</strong> ${formatFileSize(video.fileSize)}</p>
            <p><strong>Created:</strong> ${new Date(video.createdAt).toLocaleString()}</p>
            <p><strong>Updated:</strong> ${new Date(video.updatedAt).toLocaleString()}</p>
            ${video.ipfsCid ? `<p><strong>IPFS CID:</strong> ${video.ipfsCid}</p>` : ''}
        `;

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
