function SanPhamServices() {
    this.getListProductApi = function () {
        return axios({
            url: "http://localhost:8080/getAllAssets",
            method: "GET",
        });
    };

    this.addProductApi = function (product) {
        return axios({
            url: "http://localhost:8080/createAsset",
            method: "POST",
            data: product,
        });
    };
}
