(function () {
    var oldData;
    var html = '';
    html += '<a href="" class="diy export" data-type="json">导出json</a>',
    html += '<a href="" class="diy export" data-type="md">导出md</a>',
    html += '<a href="" class="diy export" data-type="km">导出km</a>',
    html += '<a href="" class="diy export" data-type="svg">导出svg</a>',
    html += '<a href="" class="diy export" data-type="txt">导出text</a>',
    html += '<a href="" class="diy export" data-type="png">导出png</a>',
    html += '<button class="diy input">',
    html += '导入<input type="file" id="fileInput" accept=".km,.txt,.md,.json" >',
    html += '</button>';

    $('.editor-title').append(html);

    $('.diy').css({
        // 'height': '30px',
        // 'line-height': '30px',
        'margin-top': '0px',
        'float': 'right',
        'background-color': '#fff',
        'min-width': '60px',
        'text-decoration': 'none',
        color: '#999',
        'padding': '0 10px',
        border: 'none',
        'border-right': '1px solid #ccc',
    });
    $('.input').css({
        'overflow': 'hidden',
        'position': 'relative',
    }).find('input').css({
        cursor: 'pointer',
        position: 'absolute',
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
        display: 'inline-block',
        opacity: 0
    });

    $(document).on('click', '.export', function (event) {
        event.preventDefault();
        var $this = $(this),
        type = $this.data('type'),
        exportType;
        switch (type) {
        case 'km':
            exportType = 'json';
            break;
        case 'md':
            exportType = 'markdown';
            break;
        case 'svg':
            exportType = 'svg';
            break;
        case 'txt':
            exportType = 'text';
            break;
        case 'png':
            exportType = 'png';
            break;
        default:
            exportType = type;
            break;
        }

        editor.minder.exportData(exportType).then(function (content) {
            switch (exportType) {
            case 'json':
                console.log($.parseJSON(content));
                break;
            default:
                console.log(content);
                break;
            }
            var blob = new Blob();
            switch (exportType) {
            case 'png':
                blob = dataURLtoBlob(content); //将base64编码转换为blob对象
                break;
            default:
                blob = new Blob([content]);
                break;
            }
            var a = document.createElement("a"); //建立标签，模拟点击下载
            a.download = $('#node_text1').text() + '.' + type;
            a.href = URL.createObjectURL(blob);
            a.click();

        });
    });

    // 导入
    window.onload = function () {
        var fileInput = document.getElementById('fileInput');

        fileInput.addEventListener('change', function (e) {
            var file = fileInput.files[0],
            // textType = /(md|km)/,
            fileType = file.name.substr(file.name.lastIndexOf('.') + 1);
            console.log(file);
            switch (fileType) {
            case 'md':
                fileType = 'markdown';
                break;
            case 'txt':
                fileType = 'text';
                break;				
            case 'km':
            case 'json':
                fileType = 'json';
                break;
            default:
                console.log("File not supported!");
                alert('只支持.km、.md、、text、.json文件');
                return;
            }
            var reader = new FileReader();
            reader.onload = function (e) {
                var content = reader.result;
                editor.minder.importData(fileType, content).then(function (data) {
                    console.log(data)
                    $(fileInput).val('');
                });
            }
            reader.readAsText(file);
        });
    }

})();

//base64转换为图片blob
function dataURLtoBlob(dataurl) {
    var arr = dataurl.split(',');
    //注意base64的最后面中括号和引号是不转译的
    var _arr = arr[1].substring(0, arr[1].length - 2);
    var mime = arr[0].match(/:(.*?);/)[1],
    bstr = atob(_arr),
    n = bstr.length,
    u8arr = new Uint8Array(n);
    while (n--) {
        u8arr[n] = bstr.charCodeAt(n);
    }
    return new Blob([u8arr], {
        type: mime
    });
}
