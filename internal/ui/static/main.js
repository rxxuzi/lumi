document.addEventListener('DOMContentLoaded', function() {
    const form = document.getElementById('lumiForm');
    const launchButton = document.getElementById('launchButton');
    const statusElement = document.getElementById('status');
    const progressContainer = document.getElementById('progress-container');
    const progressBar = document.querySelector('.progress-bar-fill');
    const progressText = document.getElementById('progress-text');

    form.addEventListener('submit', function(e) {
        e.preventDefault();

        const formData = new FormData(this);
        const jsonData = {};

        formData.forEach((value, key) => {
            if (key === 'tag' || key === 'and' || key === 'ignore') {
                jsonData[key] = value.split(',').map(item => item.trim()).filter(item => item !== '');
            } else if (key === 'mediaCount') {
                jsonData[key] = parseInt(value) || 20;
            } else {
                jsonData[key] = value;
            }
        });

        launchButton.disabled = true;
        statusElement.textContent = 'Launching Lumi...';
        progressContainer.style.display = 'block';
        progressBar.style.width = '0%';
        progressText.textContent = 'Initializing...';

        fetch('/launch', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(jsonData),
        })
            .then(response => response.text())
            .then(data => {
                updateStatus(data);
                pollStatus();
            })
            .catch((error) => {
                console.error('Error:', error);
                updateStatus('Error: ' + error.message);
                launchButton.disabled = false;
            });
    });

    function updateStatus(message) {
        statusElement.textContent = message;
    }

    function updateProgress(progress) {
        console.log("Received progress:", progress); // デバッグ情報

        const progressContainer = document.getElementById('progress-container');
        const progressBar = document.querySelector('.progress-bar-fill');
        const progressText = document.getElementById('progress-text');

        progressContainer.style.display = 'block';
        const totalImages = progress.requestedMedia;
        const downloadedImages = progress.downloadedImages;
        const skippedImages = progress.skippedImages;

        const percentage = (downloadedImages / totalImages) * 100 || 0;
        progressBar.style.width = `${percentage}%`;

        progressText.textContent = `Downloaded: ${downloadedImages}/${totalImages} (Skipped: ${skippedImages})`;
    }

    function pollStatus() {
        const interval = setInterval(() => {
            fetch('/status')
                .then(response => response.json())
                .then(data => {
                    console.log("Received status data:", data);

                    updateStatus(`Current status: ${data.status}`);

                    if (data.progress) {
                        updateProgress(data.progress);
                    }

                    if (data.status === 'Completed' || (data.progress && data.progress.terminated)) {
                        clearInterval(interval);
                        launchButton.disabled = false;
                    }
                })
                .catch(error => {
                    console.error('Error polling status:', error);
                    clearInterval(interval);
                    launchButton.disabled = false;
                });
        }, 1000);
    }
});