document.addEventListener("DOMContentLoaded", function() {
    var btn = document.getElementById("scroll-top");
    if(!btn) return;
    window.addEventListener("scroll", function() {
        if (window.scrollY > 300) btn.classList.add("visible");
        else btn.classList.remove("visible");
    });
    btn.addEventListener("click", function() {
        window.scrollTo({ top: 0, behavior: "smooth" });
    });
});
</script><div id="lab-overlay" style="display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; z-index: 99999; flex-direction: column; font-family: monospace;"><div class="lab-header" style="padding: 15px; display: flex; justify-content: space-between; align-items: center;"><h2 style="margin: 0; font-size: 18px;">Лаборатория ORCID</h2><button onclick="document.getElementById('lab-overlay').style.display='none'" style="background: #d32f2f; color: white; border: none; padding: 8px 15px; border-radius: 5px; cursor: pointer; font-weight: bold;">✕ Закрыть</button></div><div id="lab-result" style="padding: 20px; overflow-y: auto; flex: 1; font-size: 14px;"><p style="opacity: 0.7;">Выберите метод в меню...</p></div></div><script>
    </button><iframe id="en-resume-frame" src="about:blank" style="width: 100%; height: 100%; border: none; display: block;"></iframe></div><script>
    function openEnResume(e) {
        if (e) e.preventDefault();
// Создаем оверлей на лету (если его нет)
        let overlay = document.getElementById('en-resume-overlay');
        if (!overlay) {
            overlay = document.createElement('div');
            overlay.id = 'en-resume-overlay';
            overlay.style.cssText = 'display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; z-index: 999999; background: #121212;';
const btn = document.createElement('button');
            btn.textContent = '← На главную (RU)';
            btn.style.cssText = 'position: absolute; top: 15px; left: 15px; z-index: 1000000; background: #0288d1; color: #fff; border: none; padding: 10px 15px; border-radius: 8px; font-weight: bold; cursor: pointer; box-shadow: 0 4px 10px rgba(0,0,0,0.3);';
            btn.onclick = closeEnResume;
const iframe = document.createElement('iframe');
            iframe.id = 'en-resume-frame';
            iframe.style.cssText = 'width: 100%; height: 100%; border: none; display: block;';
overlay.appendChild(btn);
            overlay.appendChild(iframe);
            document.body.appendChild(overlay);
        }
overlay.style.display = 'block';
        const iframe = document.getElementById('en-resume-frame');
// Загружаем Base64
        if (window.EN_HTML_BASE64) {
            try {
                const b64Data = window.EN_HTML_BASE64.split(',')[1];
                const binaryString = atob(b64Data);
                const bytes = new Uint8Array(binaryString.length);
                for (let i = 0; i < binaryString.length; i++) {
                    bytes[i] = binaryString.charCodeAt(i);
                }
                let decodedHtml = new TextDecoder('utf-8').decode(bytes);
                decodedHtml = decodedHtml.replace('<head>', '<head><base href="./">');
const doc = iframe.contentWindow.document;
                doc.open();
                doc.write(decodedHtml);
                doc.close();
            } catch(err) {
                iframe.srcdoc = '<p style="color:red; padding: 20px;">Ошибка Base64: ' + err.message + '</p>';
            }
        } else {
            iframe.srcdoc = '<p style="color:red; padding: 20px;">Переменная EN_HTML_BASE64 не найдена.</p>';
        }
// Закрываем меню
        const nav = document.getElementById('site-navigation');
        if (nav) nav.classList.remove('is-open');
    }
function closeEnResume() {
        const overlay = document.getElementById('en-resume-overlay');
        const iframe = document.getElementById('en-resume-frame');
        overlay.style.display = 'none';
        const doc = iframe.contentWindow.document;
        doc.open();
        doc.write('');
        doc.close();
    }
</script><script src="./en_base64.js"></script><script></script><script></script><script></script><script>
</script><script></script><script>
    function openEnPage() {
        const nav = document.getElementById('site-navigation');
        if (nav) nav.classList.remove('is-open');
        const xhr = new XMLHttpRequest();
        xhr.open('GET', './en/index.html', true);
        xhr.onreadystatechange = function() {
            if (xhr.readyState === 4 && (xhr.status === 200 || xhr.status === 0)) {
                document.open();
                document.write(xhr.responseText);
                document.close();
            }
        };
        xhr.send();
    }
