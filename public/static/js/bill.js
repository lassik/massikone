$(function() {

    "use strict";

    function getImageId() {
        return $("#image-id").val();
    }

    function setImageId(imageId) {
        imageId = imageId ? imageId : "";
        var hasImage = (imageId !== "");
        $("#image-id").val(imageId);
        $("#image-upload-progress").hide();
        $("#image-select-button").show();
        $("#image-remove-button").toggle(hasImage);
        $("#image-rotate-button").toggle(hasImage);
        $("#bill-image-placeholder").toggle(!hasImage);
        $("#bill-image-container").empty();
        if (hasImage) {
            $("#bill-image-container").append(
                $("<img>", {src: "/api/userimage/"+imageId}));
        }
    }

    function uploadImage() {
        $.post({
            url: "/api/userimage",
            data: new FormData(document.getElementById("image-upload-form")),
            processData: false,
            contentType: false,
            dataType: "text",
        }).uploadProgress(function(e) {
            var percent = 0;
            if (e.lengthComputable && e.total !== 0) {
                percent = Math.round((e.loaded * 100) / e.total);
            }
            $('#image-upload-progress-bar').css('width', percent+"%")
                .attr('aria-valuenow', percent);
            $("#image-upload-progress").show();
            $("#image-select-button").hide();
            $("#image-remove-button").hide();
            $("#image-rotate-button").hide();
        }).done(function(imageId) {
            setImageId(imageId);
        }).fail(function(jqXHR) {
            setImageId(null);
            alert("Error: " + jqXHR.statusText);
        });
    }

    function rotateImage() {
        $.get({
            url: "/api/userimage/rotated/"+getImageId(),
        }).done(function(imageId) {
            setImageId(imageId);
        }).fail(function(jqXHR) {
            alert("Error: " + jqXHR.statusText);
        });
    }

    document.getElementById("image-upload-file").onchange = function() {
        uploadImage();  // Calling form submit() here doesn't work.
    };

    $("#image-upload-form").submit(function(e) {
        e.preventDefault();
        uploadImage();
    });

    $("#image-select-button").click(function(e) {
        $("#image-upload-file").trigger("click");
    });

    $("#image-remove-button").click(function(e) {
        setImageId(null);
    });

    $("#image-rotate-button").click(function(e) {
        rotateImage();
    });

    function formatEuros(euros) {
        return euros.toFixed(2).replace(".", ",");
    }

    function formatCents(cents) {
        return formatEuros(cents / 100.0);
    }

    function getCentsPerKilometer() {
        return 43;
    }

    function setPaidTypeNoUpdate(paidType) {
        paidType = paidType ? paidType : "self";
        $("#bill-form input[name=paid_type]").val([paidType]);
        return paidType;
    }

    function getPaidType() {
        return setPaidTypeNoUpdate($("#bill-form input[name=paid_type]:checked").val());
    }

    function updateCost() {
        var paidType = getPaidType();
        var isCar = (paidType === "car");
        $(".car").toggle(isCar);
        $(".notcar").toggle(!isCar);
        var unit_count = isCar ? Number($("#kilometer-input").val()) : 1;
        var unit_cost_cents = isCar ? getCentsPerKilometer() : $("#total-cost-input").val();
        var total_cost_cents = unit_count * unit_cost_cents;
        $("#euros-per-kilometer-display").text(formatCents(getCentsPerKilometer()));
        $("#total-cost-display").text(formatCents(total_cost_cents));
    }

    function setPaidType(paidType) {
        setPaidTypeNoUpdate(paidType);
        updateCost();
    }

    $("#kilometer-input").on("input", updateCost);
    $("#total-cost-input").on("input", updateCost);
    $("#bill-form input[name=paid_type]").change(updateCost);
    $("#bill-form input[name=paid_user_id]").val($("#paid-user-id-init").val());
    setPaidType($("#paid-type-init").val());
    setImageId(getImageId());

});
