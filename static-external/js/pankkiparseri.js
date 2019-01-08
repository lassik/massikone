var Pankkiparseri = {};

Pankkiparseri.withoutLeadingDayAndMonth = function(s) {
  var g = s.match(/^[0-3]\d[0-1]\d (.*)$/);
  return g ? g[1] : s;
};

Pankkiparseri.parseFinnishDate = function(finnishDate) {
  var g = finnishDate.match(/^(\d{1,2})\.(\d{1,2})\.(\d{2,4})$/);
  if (!g) {
    return null;
  }
  var year = g[3].length === 2 ? "20" + g[3] : g[3];
  var month = g[2].padStart(2, "0");
  var day = g[1].padStart(2, "0");
  return {
    finnish: finnishDate,
    iso: year + "-" + month + "-" + day,
    isoNoDashes: year + month + day
  };
};

Pankkiparseri.parseAmountPreparsed = function(eurosStr, commaCents, signStr) {
  var euros = parseInt(eurosStr.replace(/\./g, ""), 10);
  eurosStr = String(euros);
  var centsStr = (commaCents ? commaCents.slice(1) : "0").padEnd(2, "0");
  var cents = parseInt(centsStr, 10);
  var sign = signStr === "-" ? -1 : 1;
  signStr = sign < 0 ? "-" : "+";
  return {
    cents: sign * (100 * euros + cents),
    eurosDotCents: signStr + eurosStr + "." + centsStr,
    eurosCommaCents: signStr + eurosStr + "," + centsStr
  };
};

Pankkiparseri.parseAmountSignFirst = function(formattedAmount) {
  var g = formattedAmount.match(/^([+-]?)([\d.]{1,5})(,\d{1,2})?$/);
  return g ? Pankkiparseri.parseAmountPreparsed(g[2], g[3], g[1]) : null;
};

Pankkiparseri.parseAmountSignLast = function(formattedAmount) {
  var g = formattedAmount.match(/^([\d.]{1,5})(,\d{1,2})?([+-?])$/);
  return g ? Pankkiparseri.parseAmountPreparsed(g[1], g[2], g[3]) : null;
};

Pankkiparseri.parseOmaSaastopankkiTilitapahtumatCSV = function(contents) {
  // Official CSV download from online banking "tilitapahtumat" section.
  var rows = $.csv.toArrays(contents, { separator: ";" });
  var entries = [];
  for (var lineNum = 2; lineNum <= rows.length; lineNum++) {
    var row = rows[lineNum - 1];
    var entry = {};
    entry.date = Pankkiparseri.parseFinnishDate(row[0]);
    entry.otherParty = row[1];
    entry.message = row[3].match(/^'/) ? row[3].slice(1) : "";
    entry.amount = Pankkiparseri.parseAmountSignFirst(row[4]);
    entries.push(entry);
  }
  return entries;
};

Pankkiparseri.parseOsuuspankkiTilitapahtumatCSV = function(contents) {
  // Official CSV download from online banking "tilitapahtumat" section.
  [
    "Tilitapahtumat",
    "Siirry tiliotteeseen",
    "Hae tapahtumat",
    "Lataa pelkät tilitapahtumat tiedostona (csv)"
  ];
  var rows = $.csv.toArrays(contents, { separator: ";" });
  var entries = [];
  for (var lineNum = 2; lineNum <= rows.length; lineNum++) {
    var row = rows[lineNum - 1];
    var entry = {};
    entry.date = Pankkiparseri.parseFinnishDate(row[1]);
    entry.otherParty = row[5];
    entry.message = row[4] + " " + row[8];
    entry.amount = Pankkiparseri.parseAmountSignFirst(row[2]);
    entry.referenceNumber = row[7];
    entry.archivalId = row[9];
    entries.push(entry);
  }
  return entries;
};

Pankkiparseri.parseSPankkiTilioteTabulaCSV = function(contents) {
  // CSV ripped from S-Pankki bank statement PDF by Tabula with autodetect.
  var rows = $.csv.toArrays(contents);
  var entries = [];
  var entry = null;
  var sinceEntry = null;
  var date = null;
  var g;
  for (var lineNum = 1; lineNum <= rows.length; lineNum++) {
    var row = rows[lineNum - 1];
    if ((g = row[0].match(/^KIRJAUSPÄIVÄ (\d{2}\.\d{2}\.\d{2})/))) {
      date = Pankkiparseri.parseFinnishDate(g[1]);
      entry = sinceEntry = null;
    } else if (row[0].match(/^\d{18}( [A-Z])?/)) {
      sinceEntry = 0;
      entry = {};
      entry.date = date;
      entry.archivalId = row[0];
      entry.otherParty = Pankkiparseri.withoutLeadingDayAndMonth(row[1]);
      entry.amount = Pankkiparseri.parseAmountSignLast(row[row.length - 1]);
      entry.message = "";
      entries.push(entry);
    } else if (sinceEntry === 1 || sinceEntry === 2) {
      if (entry.message.length > 0) {
        entry.message += " ";
      }
      entry.message += Pankkiparseri.withoutLeadingDayAndMonth(row[1]);
    }
    if (sinceEntry !== null) {
      sinceEntry += 1;
    }
  }
  return entries;
};

Pankkiparseri.parseOfxXml = function(contents) {
  var parser = new DOMParser();
  var doc = parser.parseFromString(contents, "text/xml");
  var stmttrn_list = doc
    .getElementsByTagName("OFX")[0]
    .getElementsByTagName("BANKMSGSRSV1")[0]
    .getElementsByTagName("STMTTRNRS")[0]
    .getElementsByTagName("STMTRS")[0]
    .getElementsByTagName("BANKTRANLIST")[0].childNodes;
  var ans = [];
  stmttrn_list.forEach(function(stmttrn) {
    var dtposted = stmttrn.getElementsByTagName("DTPOSTED")[0].childNodes[0]
      .nodeValue;
    var trnamt = stmttrn.getElementsByTagName("TRNAMT")[0].childNodes[0]
      .nodeValue;
    var memo = stmttrn.getElementsByTagName("MEMO")[0].childNodes[0].nodeValue;
    ans.push({
      Date: dtposted.replace(/^(\d{4})(\d{2})(\d{2})$/, "$1-$2-$3"),
      Cents: Math.abs(parseInt(trnamt.replace(".", ""))),
      Description: memo
    });
  });
  return ans;
};

Pankkiparseri.handleFiles = function(files, entriesCallback, parse, encoding) {
  for (var i = 0; i < files.length; i++) {
    var file = files[i];
    if (!/^text\//.test(file.type)) {
      continue;
    }
    var reader = new FileReader();
    reader.onload = function(e) {
      entriesCallback(parse(e.target.result));
    };
    reader.readAsText(file, encoding);
  }
};

Pankkiparseri.addBankToForm = function(form, entriesCallback, parse, encoding) {
  var hiddenInput = document.createElement("input");
  hiddenInput.style.display = "none";
  hiddenInput.type = "file";
  hiddenInput.accept = ".csv,text/csv";
  hiddenInput.multiple = true;
  hiddenInput.addEventListener("change", function(e) {
    Pankkiparseri.handleFiles(this.files, entriesCallback, parse, encoding);
  });
  form.appendChild(hiddenInput);
  return function(e) {
    e.preventDefault();
    hiddenInput.click();
  };
};

Pankkiparseri.addBankButtonToForm = function(
  form,
  entriesCallback,
  bankTitle,
  parse,
  encoding
) {
  var onClick = Pankkiparseri.addBankToForm(
    form,
    entriesCallback,
    parse,
    encoding
  );
  var button = document.createElement("button");
  button.appendChild(document.createTextNode(bankTitle));
  button.addEventListener("click", onClick, false);
  form.appendChild(button);
};

Pankkiparseri.formatsList = [
  {
    bankTitle: "Oma Säästöpankki",
    subtitle: "tilitapahtumat CSV",
    parse: Pankkiparseri.parseOmaSaastopankkiTilitapahtumatCSV,
    encoding: "ISO-8859-15"
  },
  {
    bankTitle: "Osuuspankki",
    subtitle: "tilitapahtumat CSV",
    parse: Pankkiparseri.parseOsuuspankkiTilitapahtumatCSV,
    encoding: "ISO-8859-15"
  },
  {
    bankTitle: "S-Pankki",
    subtitle: "tiliote Tabula CSV",
    parse: Pankkiparseri.parseSPankkiTilioteTabulaCSV,
    encoding: "UTF-8"
  }
];

Pankkiparseri.addToForm = function(formId, entriesCallback) {
  var form = document.getElementById(formId);
  for (var i = 0; i < Pankkiparseri.formatsList.length; i++) {
    var format = Pankkiparseri.formatsList[i];
    Pankkiparseri.addBankButtonToForm(
      form,
      entriesCallback,
      format.bankTitle,
      format.parse,
      format.encoding
    );
  }
};

Pankkiparseri.ofxStringFromEntries = function(entries) {
  var doc;
  function elem(tag, bodyText) {
    var e = doc.createElement(tag);
    e.append(String(bodyText));
    return e;
  }
  doc = document.implementation.createDocument(null, "OFX", null);
  var bankmsgsrsv1 = doc.createElement("BANKMSGSRSV1");
  doc.documentElement.appendChild(bankmsgsrsv1);
  var stmttrnrs = doc.createElement("STMTTRNRS");
  bankmsgsrsv1.appendChild(stmttrnrs);
  var stmtrs = doc.createElement("STMTRS");
  stmttrnrs.appendChild(stmtrs);
  var bankacctfrom = doc.createElement("BANKACCTFROM");
  bankacctfrom.appendChild(elem("BANKID", "000000000"));
  bankacctfrom.appendChild(elem("ACCTID", "0000000"));
  bankacctfrom.appendChild(elem("ACCTTYPE", "CHECKING"));
  stmtrs.appendChild(bankacctfrom);
  var banktranlist = doc.createElement("BANKTRANLIST");
  var i = 0;
  entries.forEach(function(entry) {
    var stmttrn = doc.createElement("STMTTRN");
    stmttrn.appendChild(elem("FITID", i));
    stmttrn.appendChild(elem("TRNTYPE", "CHECK"));
    stmttrn.appendChild(elem("DTPOSTED", entry.date.isoNoDashes));
    stmttrn.appendChild(elem("TRNAMT", entry.amount.eurosDotCents));
    stmttrn.appendChild(elem("MEMO", entry.message));
    banktranlist.appendChild(stmttrn);
    i++;
  });
  stmtrs.appendChild(banktranlist);
  return new XMLSerializer().serializeToString(doc);
};
