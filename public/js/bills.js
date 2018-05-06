$(function() {

    "use strict";

    $("#logout-button").click(function(e) {
        $("#logout-form").submit();
    });

    $(".clickable-row").click(function() {
        window.location = $(this).data("href");
    });

    $(".tag-toggle").click(function() {
        var tag_index = $(this).val();
        $(".tag"+tag_index).toggle();
    });

});
