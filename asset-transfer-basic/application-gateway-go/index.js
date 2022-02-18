var service = new SanPhamServices();
function getEle(id) {
    return document.getElementById(id);
}
function getData() {
    service
        .getListProductApi()
        .then(function (result) {
            renderListProduct(result.data);
            timKiem(result.data)
        })
        .catch(function (error) {
            console.log(error);
        });
}
getData();
function renderListProduct(list) {
    var contentHTML = "";

    list.forEach(function (product, index) {
        contentHTML += `
            <tr>
                <td>${index + 1}</td>
                <td>${product.AppraisedValue}</td>
                <td>${product.Color}</td>
                <td>
                ${product.ID}
                </td>
                <td>${product.Owner}</td>
                <td>${product.Size}</td>
                <td>
                    <button class="btn btn-info" data-toggle="modal" data-target="#myModal" onclick="suaSanPham(${product.id
            })">Sửa</button>
                    <button class="btn btn-danger" onclick="xoaSanPham(${product.id
            })">Xóa</button>
                </td>
            <tr>
        `;
    });

    document.getElementById("tblDanhSachSP").innerHTML = contentHTML;
}
getEle("btnThemSP").addEventListener("click", function () {
    document.getElementsByClassName("modal-title")[0].innerHTML =
        "Thêm Sản Phẩm";

    var footer =
        '<button class="btn btn-success" onclick="addProduct()">Thêm SP</button>';
    document.getElementsByClassName("modal-footer")[0].innerHTML = footer;
});
function addProduct() {
    /**
     * Dom lấy value từ các thẻ input
     */
    var AppraisedValue = getEle("AppraisedValue").value;
    var Color = getEle("Color").value;
    var ID = getEle("ID").value;
    var Owner = getEle("Owner").value;
    var Size = getEle("Size").value;
    console.log(AppraisedValue, Color, ID, Owner, Size);
    AppraisedValue = parseInt(AppraisedValue)
    Size = parseInt(Size)
    var sanPham = new SanPham(AppraisedValue, Color, ID, Owner, Size);

    service
        .addProductApi(sanPham)
        .then(function (result) {
            console.log(result);
            //Tắt modal
            document.getElementsByClassName("close")[0].click();
            //làm mới lại dữ liệu
            getData();

        })
        .catch(function (error) {
            console.log(error);
        });
}

