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

    $("#tags-modal").on("shown.bs.modal", function () {
        $.get({
            url: "/api/tags"
        }).done(function(tags) {
            $("#tag-list").empty();
            tags.forEach(function (tag) {
                var li = $("<li>").appendTo($("#tag-list"));
                li.append(tag.tag);
                li.append($('<button class="btn btn-info" type="button"><span class="glyphicon glyphicon-remove"></span></button>'));
            });
        }).fail(function(jqXHR) {
            alert("Error: " + jqXHR.statusText);
        });
    });

});
