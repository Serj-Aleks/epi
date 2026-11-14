// script/infra/ptype.js — элементы страницы, константы и состояние панели
const log = document.getElementById("log"), inp = document.getElementById("i");
const sel = document.getElementById("seans"), note = document.getElementById("note");
const tsel = document.getElementById("thema"), cardBox = document.getElementById("card");
// граф nexus: размеры узлов и промежутков
const NXG = { padding: 24, nodeW: 80, nodeH: 40, vSpacing: 10, hSpacing: 32, groupGap: 24, collectorGap: 24 };
const SVGNS = "http://www.w3.org/2000/svg";
// связи с прямым и обратным списком (линза ⊞)
const linkKeys = ["Taxon", "Links", "Attrib", "Preposit", "Prfix", "Sufix", "Radix"];
// лента: история ввода, сеанс открыт для записи, последнее состояние сервера
let hist = [], hpos = 0, writable = false, last = null;
// режим строки ввода: new, rename, clone, cnew, concept
let mode = null;
// ответ на вопрос trans — имя концепта
let conceptDone = null;
// линзы карточки: включена или нет
const lens = { graph: true, taxon: false, rank: false, tabs: false, process: false, content: true, slots: false };
// карточка: имя, таймер, источник клона, поколение сборки, найденное внутри концепта { name, ext, q }
let cardName = null, cardTimer = null, cloneSrc = "", cardGen = 0, cardFind = null;
// история панели (←) и ключ открытого списка qwest
let pstack = [], qwestKey = null;
// фон дня недели: класс body по номеру дня (0 — воскресенье)
const days = ["day-sunday", "day-monday", "day-tuesday", "day-wednesday", "day-thursday", "day-friday", "day-saturday"];
