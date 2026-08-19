// API client mirroring @animegarden/client: URL serialization/parsing and
// fetch helpers with the exact original semantics.

export const DefaultPageSize = 100;
export const MaxRequestPageSize = 1000;
export const DefaultBaseURL = '/';

export const SupportProviders = ['dmhy', 'moe', 'mikan', 'ani'] as const;
export const SupportPresets = ['bangumi'] as const;

export type ProviderType = (typeof SupportProviders)[number];
export type PresetType = (typeof SupportPresets)[number];

export interface Resource<T extends { tracker?: boolean; metadata?: boolean } = {}> {
  id: number;
  provider: string;
  providerId: string;
  title: string;
  href: string;
  type: string;
  magnet: string;
  tracker: T['tracker'] extends true ? string : string | null | undefined;
  size: number;
  fansub?: { id: number; name: string; avatar?: string } | null;
  publisher: { id: number; name: string; avatar?: string };
  subjectId?: number | null;
  createdAt: Date;
  fetchedAt: Date;
  metadata?: T['metadata'] extends true ? any : any;
}

export interface ResourceDetail {
  description: string;
  files: Array<{ name: string; size: string }>;
  magnets: Array<{ name: string; url: string }>;
  hasMoreFiles: boolean;
}

export interface Subject {
  id: number;
  name: string;
  keywords: string[];
  activedAt: Date;
  isArchived: boolean;
}

export interface FilterOptions {
  provider?: string;
  duplicate?: boolean;
  after?: Date;
  before?: Date;
  search?: string | string[];
  include?: string | string[];
  keywords?: string | string[];
  exclude?: string | string[];
  type?: string;
  types?: string[];
  subject?: number;
  subjects?: number[];
  fansub?: string;
  fansubs?: string[];
  publisher?: string;
  publishers?: string[];
  preset?: string;
}

export interface ResolvedFilterOptions {
  preset?: PresetType;
  provider?: ProviderType;
  duplicate?: boolean;
  types?: string[];
  after?: Date;
  before?: Date;
  fansubs?: string[];
  publishers?: string[];
  subjects?: number[];
  search?: string[];
  include?: string[];
  keywords?: string[];
  exclude?: string[];
}

export interface PaginationOptions {
  page?: number;
  pageSize?: number;
}

export interface ResolvedPaginationOptions {
  page: number;
  pageSize: number;
}

// --- normalizeTitle (simptrad) ---
// Ported: fullToHalf(tradToSimple(title), { punctuation: true })

const dictSimple =
  "出饥才制皑蔼碍爱肮翱袄奥坝罢摆败颁办绊帮绑镑谤剥饱宝报鲍辈贝钡狈备惫绷笔毕毙币闭辟边编贬变辩辫标鳖别瘪濒滨宾摈饼并拨钵铂驳卜补财采参蚕残惭惨灿苍舱仓沧厕侧册测层"
  "诧搀掺蝉馋谗缠铲产阐颤场尝长偿肠厂畅钞车彻尘陈衬撑称惩诚骋痴迟驰耻齿炽冲虫宠畴踌筹绸丑橱厨锄雏础储触处传疮床闯创锤唇纯绰辞词赐聪葱囱从丛凑蹿窜错达带贷担单郸掸胆"
  "惮诞弹当挡党荡档捣岛祷导盗灯邓敌涤递缔颠点垫电淀钓调迭谍叠钉顶锭订丢东动栋冻斗犊独读赌镀锻断缎兑队对吨顿钝夺堕鹅额讹恶饿儿尔饵贰发罚阀珐矾钒烦范贩饭访纺飞诽废费"
  "纷坟奋愤粪丰枫峰锋风疯冯缝讽凤肤辐抚辅赋复负讣妇缚该钙盖干赶秆赣冈刚钢纲岗杠皋镐搁鸽阁铬个给龚宫巩贡钩沟构购够蛊顾雇剐挂关观馆惯贯广规硅归龟闺轨诡柜贵刽辊滚锅国"
  "过骇韩汉号阂鹤贺横恒轰鸿红后壶护沪户哗华画划话怀坏欢环还缓换唤痪焕涣黄谎挥辉毁贿秽会烩汇讳诲绘荤浑伙获货祸击机积饥迹讥鸡绩缉极辑级挤几蓟剂济计记际继纪夹荚颊贾钾"
  "价驾歼监坚笺间艰缄茧检碱硷拣捡简俭减荐槛鉴践贱见键舰剑饯渐溅涧将浆蒋桨奖讲酱胶浇骄娇搅铰矫侥脚饺缴绞轿较秸阶节茎鲸惊经颈静镜径痉竞净纠厩旧驹举据锯惧剧鹃绢杰洁结"
  "诫届紧锦仅谨进晋烬尽劲荆觉决诀绝钧军骏开凯颗壳课垦恳抠库裤夸块侩宽矿旷况亏岿窥馈溃扩阔蜡腊莱来赖蓝栏拦篮阑兰澜谰揽览懒缆烂滥捞劳涝乐镭垒类泪厘篱离里鲤礼栗丽厉励"
  "砾历沥隶俩联莲连镰怜涟帘敛脸链恋炼练粮凉两辆谅疗辽镣猎临邻鳞凛赁龄铃凌灵岭领馏刘龙聋咙笼垄拢陇楼娄搂篓芦卢颅庐炉掳卤虏鲁赂禄录陆驴吕铝侣屡缕虑滤绿峦挛孪滦乱抡轮"
  "伦仑沦纶论萝罗逻锣箩骡骆络妈玛码蚂马骂吗买麦卖迈脉瞒馒蛮满谩猫锚铆贸么霉没镁门闷们锰梦谜弥秘觅幂绵缅庙灭悯闽鸣铭谬谋亩呐钠纳难挠脑恼闹馁内拟你腻撵捻酿鸟聂啮镊镍"
  "柠狞宁拧泞钮纽脓浓农疟诺欧鸥殴呕沤盘庞抛赔喷鹏骗飘频贫苹凭评泼颇扑铺仆朴谱栖凄脐齐骑岂启气弃讫牵扦钎铅迁签谦钱钳潜浅谴堑枪呛墙蔷强抢锹桥乔侨翘窍窃钦亲寝轻氢倾顷"
  "请庆琼穷趋区躯驱龋颧权劝却鹊确群让饶扰绕热韧认纫荣绒软锐闰润洒萨鳃赛伞丧骚扫涩杀刹纱筛晒删闪陕赡缮伤赏烧绍赊摄慑设绅审婶肾渗声绳胜圣师狮湿诗尸虱时蚀实识驶势适释"
  "饰视试寿兽枢输书赎属术树竖数帅双谁税顺说硕烁丝饲松耸怂颂讼诵擞苏诉肃虽随绥岁孙损笋缩琐锁獭挞抬台态摊贪瘫滩坛谭谈叹汤烫涛绦讨腾誊锑题体屉条贴铁厅听烃铜统头秃图涂"
  "团颓蜕托脱鸵驮驼椭洼袜弯湾顽万网韦违围为潍维苇伟伪纬谓卫温闻纹稳问瓮挝蜗涡窝卧呜钨乌污诬无芜吴坞雾务误锡牺袭习铣戏细虾辖峡侠狭厦吓锨鲜纤咸贤衔闲显险现献县馅羡宪"
  "线厢镶乡详响项萧嚣销晓啸蝎协挟携胁谐写泻谢锌衅兴凶汹锈绣虚嘘须许叙绪续轩悬选癣绚学勋熏询寻驯训讯逊压鸦鸭哑亚讶阉烟盐严岩颜阎艳厌砚彦谚验鸯杨扬疡阳痒养样瑶摇尧遥"
  "窑谣药爷页业叶医铱颐遗仪彝蚁艺亿忆义诣议谊译异绎荫阴银饮隐樱婴鹰应缨莹萤营荧蝇赢颖哟拥佣痈踊咏涌优忧邮铀犹游诱于舆余鱼渔娱与屿语郁吁御狱誉预驭鸳渊辕园员圆缘远愿"
  "约跃钥岳粤悦阅云郧匀陨运蕴酝晕韵杂灾载攒暂赞赃脏凿枣皂灶责择则泽贼赠扎札轧铡闸栅诈斋债毡盏斩辗崭栈占战绽张涨帐账胀赵蛰辙锗这贞针侦诊镇阵挣睁征狰争帧郑证织职执纸"
  "挚掷帜质滞钟终种肿众诌轴皱昼骤猪诸诛烛瞩嘱贮铸筑驻专砖转赚桩庄装妆壮状锥赘坠缀谆准着浊兹咨资渍踪综总纵邹诅组钻闩刍劢叽戋讦讧讪邝钅亘伛伥伧伫凫厍圹忏扪犷犸玑纡纣"
  "纥纨纩芗讴讵讷邬钆钇闫饧佥刭吣呒呓呖呗呙囵坂坜奁奂妩妪妫岖岘岙岚帏庑忾怃怄怅怆抟杩欤沣沩炀疖矶纭纰纾芈苁苈苋苌苎虬诂诃诋诎诏诒轫邺钊钋钌闱闳闵闶陉饨饩饪饫饬鸠侪"
  "侬兖刿剀匦卺咛咝垅垆姗岽峄弪怿戗昙枞枥枧枨枭殁泷泸泺泾炖炜炝牦玮瓯疠砀籴绀绁绂绉绋绌绐肴苘茏茑茔茕虮诓诔诖诘诙诜诟诠诤诨诩轭迩迳郏郐郓钍钏钐钔钕钗饴驵驷驸驺驽驿"
  "骀鸢黾祎俣俦俨俪咤哒哓哔哕哙哜哝垩垭垲垴姹娅娆娈峤峥怼恸恹恺恻恽挢昵柽栀栉栊栌栎殇泶浃浈浍浏浒浔狯狲珑疬眍砗砜祢笃籼绔绗绛胧胨胪胫舣荛荜荞荟荠荥荦荨荩荪荬荭荮虿"
  "觇诮诰诳诶贲贳贶贻轱轲轳轵轶轷轸轹轺郦钚钛钜钣钤钪钫钬钭钯闼闾陧顸飑飒饷骁骅骈鸨鸩袅唛唠唢埘埙埚娲娴崂崃帱徕悭晔晖栾桠桡桢桤桦桧氩涞涠烨猃玺珲疱疴砺砻祯笕绠绡绨"
  "脍莅莳莴莶莸莺莼蚝蚬衮觊诹诼诿谀谂谄谇贽赀赅赆趸轼轾辁辂逦钰钲钴钶钷钸钹钺钼钽钿铄铈铉铊铋铌铍铎阃阄阆隽顼颀颃饽馀骊鸪鸫鸬鸱鸲鸶龀龛偬偻偾匮厣啧啬啭埯婵帻帼悫惬"
  "掴掼棂殒殓渌渎渑渖焖焘猕猡琏痖皲眦硖硗稆笾粜粝绫绮绯绱绲绶绺绻绾缁缍羟聍脶舻萦蛎蛏裆觋谌谏谑谒谔谕谖谘谙谛谝赇赈赉跄辄铐铑铒铕铖铗铘铙铛铞铟铠铢铤铥铧铨铩铪铫铮"
  "铯铳铴铵铷阈阊阋阌阍阏馄骐骒骓骖鸷鸸鸹鸺鸾麸亵傥傧傩喽喾媪嵘嵝巯弑愠愦揿椁椟椠椤殚毵溆牍猬痨痫睐睑禅筚筝絷缂缃缇缈缋缌缏缑缒缗脔腌蒇蒉蒌蛱蛲蛳蛴裢裣裥觌觞谟谠谡"
  "谥谧赍赓赕跞辇辋辍辎铹铼铽铿锂锃锆锇锉锊锍锎锏锒锓锔锕阒阕雳靓颉颌颍颏飓飨馇馊骘骛鱿鲂鹁鹂鹄鹆鹇鹈鼋缙嗫嗳嫒嫔尴摅榄榇榈榉毂氲滗滟滠滢瘅碛碜禀稣窦缛缜缟缡缢缣缤"
  "耢腭腼腽蓠蓣蓥蓦觎谪谫跷跸跹跻辏辔锖锘锛锝锞锟锢锩锪锫锬锱阖阗阙韪韫颔飕馍馐骜骝骞骟鲅鲆鲇鲈鲋鲎鲐鹉鹋鹌鹎鹑龃龅龆厮嘤嫱戬撄暧槟槠殡潆潇潋潴瑷瘗瘘窭箦箧箨箪箫粽"
  "糁缥缦缧缪缫罂罴膑蔹蔺蝈褛觏谮谯谲赙酽酾銮锲锴锵锶锷锸锺锼锾锿镂镄镅阚霁韬馑骠骢鲑鲒鲔鲕鲚鲛鲞鲟鹕鹗鹘鹚鹛鹜麽龇龈噜屦幞撷撸撺樯橥璎篑糇糍缬缭缯耧聩蕲蝼蝾褴觐觑"
  "觯谳谵赜踬踯辘镆镉镌镎镏镒镓镔靥鞑鞒颚颛餍馓馔骣魇鲠鲡鲢鲣鲥鲦鲧鲨鲩鲫鹞鹣齑龉龊廪懔斓橹橼氇濑瘾瘿穑缰缱缲缳薮螨赝辚錾镖镗镘镙镛镝镞镟颞颟颡飙飚魉鲭鲮鲰鲱鲲鲳鲴"
  "鲵鲶鲷鲺鲻鹦鹧鹨鹾黉嬷懑檩簖羁膻藓蹑蹒镡镢镤镥镦镧镨镩镪镫鲼鲽鳄鳅鳆鳇鳊鳋鹩鹪鹫鹬龌冁癞镬镯镱雠鞯颢髅鳌鳍鳎鳏鳐鹭鹱巅籁缵谶蹰镲霭鞲骥髋髌鳓鳔鳕鳗鳘鳙鼗瓒蘖镳颥"
  "骧鬓鳜鳝鳟黩黪鼍灏癫躏颦鳢鹳趱躜鼹齄馕戆里回赞砖厂乃千干为升厅历夫扎冬只台处奶布仿众伪关冱凼吊吃合向回奸尽岁戏当扣曲朱欢杂纤考阶贠佑体佛克刨医呆听困坑址坛坝妫局"
  "奁妩彷志折杆沩泛灵系芸苏证谷闲韧驱旸佩玙侄凭刮厕卷周咔国岩岽幸弦念征抵拐杯昆构板欣泄沓注沾牦狍线罗舍肷规表视迤郄采隶闸闹举兹修勋咴叙咱咽哄哗垧姜恤总恹珏珉竖绒秋"
  "背胡荡荫药蚁钟钥钩面浐浕饸饹俯闿冢埙娘唣家席恶挽捆效核桊殷涂浚留狸症绣脏艳莜莅致莼赃赆钵钻铁验偷鸮勖啖啮庵彩惭戚欲渎焊略球琅盘眺硗笺累绩绱绷菱衔谌酝铲馆麻硚谞琎"
  "啰馃傈剩堤喂嵛揸揿搜棱棰椟棹焰畲缗舄裥赍跖辉铺锄锈锎锏阔骗馈鹀剿翚叠尴慑愍愈携暖溜滟溪漓煅照碜痹筱签蒙蓑辞酬鉴酰锨颓跶墙墉榨旗演模槔璃瘘碱端管熏碹箬罂蔑蜷酸锹镅"
  "嘻璇澄觑镎飘鲠鲧镕噪懒燕赞赝雕鲶赟檐磷襁镢鳅鳄嚣藤翻镯鬃攒耀鬓鳝癯";

const dictTrad =
  "齣饑纔製皚藹礙愛骯翺襖奧壩罷擺敗頒辦絆幫綁鎊謗剝飽寶報鮑輩貝鋇狽備憊繃筆畢斃幣閉闢邊編貶變辯辮標鱉別癟瀕濱賓擯餅並撥鉢鉑駁蔔補財採參蠶殘慚慘燦蒼艙倉滄廁側冊測層"
  "詫攙摻蟬饞讒纏鏟產闡顫場嘗長償腸廠暢鈔車徹塵陳襯撐稱懲誠騁癡遲馳恥齒熾沖蟲寵疇躊籌綢醜櫥廚鋤雛礎儲觸處傳瘡牀闖創錘脣純綽辭詞賜聰蔥囪從叢湊躥竄錯達帶貸擔單鄲撣膽"
  "憚誕彈當擋黨蕩檔搗島禱導盜燈鄧敵滌遞締顛點墊電澱釣調叠諜疊釘頂錠訂丟東動棟凍鬥犢獨讀賭鍍鍛斷緞兌隊對噸頓鈍奪墮鵝額訛惡餓兒爾餌貳發罰閥琺礬釩煩範販飯訪紡飛誹廢費"
  "紛墳奮憤糞豐楓峯鋒風瘋馮縫諷鳳膚輻撫輔賦復負訃婦縛該鈣蓋幹趕稈贛岡剛鋼綱崗槓臯鎬擱鴿閣鉻個給龔宮鞏貢鉤溝構購夠蠱顧僱剮掛關觀館慣貫廣規矽歸龜閨軌詭櫃貴劊輥滾鍋國"
  "過駭韓漢號閡鶴賀橫恆轟鴻紅後壺護滬戶譁華畫劃話懷壞歡環還緩換喚瘓煥渙黃謊揮輝毀賄穢會燴匯諱誨繪葷渾夥獲貨禍擊機積飢跡譏雞績緝極輯級擠幾薊劑濟計記際繼紀夾莢頰賈鉀"
  "價駕殲監堅箋間艱緘繭檢鹼礆揀撿簡儉減薦檻鑑踐賤見鍵艦劍餞漸濺澗將漿蔣槳獎講醬膠澆驕嬌攪鉸矯僥腳餃繳絞轎較稭階節莖鯨驚經頸靜鏡徑痙競淨糾廄舊駒舉據鋸懼劇鵑絹傑潔結"
  "誡屆緊錦僅謹進晉燼盡勁荊覺決訣絕鈞軍駿開凱顆殼課墾懇摳庫褲誇塊儈寬礦曠況虧巋窺饋潰擴闊蠟臘萊來賴藍欄攔籃闌蘭瀾讕攬覽懶纜爛濫撈勞澇樂鐳壘類淚釐籬離裏鯉禮慄麗厲勵"
  "礫歷瀝隸倆聯蓮連鐮憐漣簾斂臉鏈戀煉練糧涼兩輛諒療遼鐐獵臨鄰鱗凜賃齡鈴淩靈嶺領餾劉龍聾嚨籠壟攏隴樓婁摟簍蘆盧顱廬爐擄滷虜魯賂祿錄陸驢呂鋁侶屢縷慮濾綠巒攣孿灤亂掄輪"
  "倫侖淪綸論蘿羅邏鑼籮騾駱絡媽瑪碼螞馬罵嗎買麥賣邁脈瞞饅蠻滿謾貓錨鉚貿麼黴沒鎂門悶們錳夢謎彌祕覓冪綿緬廟滅憫閩鳴銘謬謀畝吶鈉納難撓腦惱鬧餒內擬妳膩攆撚釀鳥聶齧鑷鎳"
  "檸獰寧擰濘鈕紐膿濃農瘧諾歐鷗毆嘔漚盤龐拋賠噴鵬騙飄頻貧蘋憑評潑頗撲鋪僕樸譜棲悽臍齊騎豈啓氣棄訖牽扡釺鉛遷籤謙錢鉗潛淺譴塹槍嗆牆薔強搶鍬橋喬僑翹竅竊欽親寢輕氫傾頃"
  "請慶瓊窮趨區軀驅齲顴權勸卻鵲確羣讓饒擾繞熱韌認紉榮絨軟銳閏潤灑薩鰓賽傘喪騷掃澀殺剎紗篩曬刪閃陝贍繕傷賞燒紹賒攝懾設紳審嬸腎滲聲繩勝聖師獅溼詩屍蝨時蝕實識駛勢適釋"
  "飾視試壽獸樞輸書贖屬術樹豎數帥雙誰稅順說碩爍絲飼鬆聳慫頌訟誦擻蘇訴肅雖隨綏歲孫損筍縮瑣鎖獺撻擡臺態攤貪癱灘壇譚談嘆湯燙濤絛討騰謄銻題體屜條貼鐵廳聽烴銅統頭禿圖塗"
  "團頹蛻託脫鴕馱駝橢窪襪彎灣頑萬網韋違圍爲濰維葦偉僞緯謂衛溫聞紋穩問甕撾蝸渦窩臥嗚鎢烏汙誣無蕪吳塢霧務誤錫犧襲習銑戲細蝦轄峽俠狹廈嚇杴鮮纖鹹賢銜閒顯險現獻縣餡羨憲"
  "線廂鑲鄉詳響項蕭囂銷曉嘯蠍協挾攜脅諧寫瀉謝鋅釁興兇洶鏽繡虛噓須許敘緒續軒懸選癬絢學勳薰詢尋馴訓訊遜壓鴉鴨啞亞訝閹煙鹽嚴巖顏閻豔厭硯彥諺驗鴦楊揚瘍陽癢養樣瑤搖堯遙"
  "窯謠藥爺頁業葉醫銥頤遺儀彜蟻藝億憶義詣議誼譯異繹蔭陰銀飲隱櫻嬰鷹應纓瑩螢營熒蠅贏穎喲擁傭癰踴詠湧優憂郵鈾猶遊誘於輿餘魚漁娛與嶼語鬱籲禦獄譽預馭鴛淵轅園員圓緣遠願"
  "約躍鑰嶽粵悅閱雲鄖勻隕運蘊醞暈韻雜災載攢暫贊贓髒鑿棗皁竈責擇則澤賊贈紮劄軋鍘閘柵詐齋債氈盞斬輾嶄棧佔戰綻張漲帳賬脹趙蟄轍鍺這貞針偵診鎮陣掙睜徵猙爭幀鄭證織職執紙"
  "摯擲幟質滯鍾終種腫衆謅軸皺晝驟豬諸誅燭矚囑貯鑄築駐專磚轉賺樁莊裝妝壯狀錐贅墜綴諄準著濁茲諮資漬蹤綜總縱鄒詛組鑽閂芻勱嘰戔訐訌訕鄺釒亙傴倀傖佇鳧厙壙懺捫獷獁璣紆紂"
  "紇紈纊薌謳詎訥鄔釓釔閆餳僉剄唚嘸囈嚦唄咼圇阪壢奩奐嫵嫗嬀嶇峴嶴嵐幃廡愾憮慪悵愴摶榪歟灃潙煬癤磯紜紕紓羋蓯藶莧萇苧虯詁訶詆詘詔詒軔鄴釗釙釕闈閎閔閌陘飩餼飪飫飭鳩儕"
  "儂兗劌剴匭巹嚀噝壠壚姍崬嶧弳懌戧曇樅櫪梘棖梟歿瀧瀘濼涇燉煒熗犛瑋甌癘碭糴紺紲紱縐紼絀紿餚檾蘢蔦塋煢蟣誆誄詿詰詼詵詬詮諍諢詡軛邇逕郟鄶鄆釷釧釤鍆釹釵飴駔駟駙騶駑驛"
  "駘鳶黽禕俁儔儼儷吒噠嘵嗶噦噲嚌噥堊埡塏堖奼婭嬈孌嶠崢懟慟懨愷惻惲撟暱檉梔櫛櫳櫨櫟殤澩浹湞澮瀏滸潯獪猻瓏癧瞘硨碸禰篤秈絝絎絳朧腖臚脛艤蕘蓽蕎薈薺滎犖蕁藎蓀蕒葒葤蠆"
  "覘誚誥誑誒賁貰貺貽軲軻轤軹軼軤軫轢軺酈鈈鈦鉅鈑鈐鈧鈁鈥鈄鈀闥閭隉頇颮颯餉驍驊駢鴇鴆嫋嘜嘮嗩塒壎堝媧嫺嶗崍幬徠慳曄暉欒椏橈楨榿樺檜氬淶潿燁獫璽琿皰痾礪礱禎筧綆綃綈"
  "膾蒞蒔萵薟蕕鶯蓴蠔蜆袞覬諏諑諉諛諗諂誶贄貲賅贐躉軾輊輇輅邐鈺鉦鈷鈳鉕鈽鈸鉞鉬鉭鈿鑠鈰鉉鉈鉍鈮鈹鐸閫鬮閬雋頊頎頏餑餘驪鴣鶇鸕鴟鴝鷥齔龕傯僂僨匱厴嘖嗇囀垵嬋幘幗愨愜"
  "摑摜櫺殞殮淥瀆澠瀋燜燾獼玀璉瘂皸眥硤磽穭籩糶糲綾綺緋鞝緄綬綹綣綰緇綞羥聹腡艫縈蠣蟶襠覡諶諫謔謁諤諭諼諮諳諦諞賕賑賚蹌輒銬銠鉺銪鋮鋏鋣鐃鐺銱銦鎧銖鋌銩鏵銓鎩鉿銚錚"
  "銫銃鐋銨銣閾閶鬩閿閽閼餛騏騍騅驂鷙鴯鴰鵂鸞麩褻儻儐儺嘍嚳媼嶸嶁巰弒慍憒撳槨櫝槧欏殫毿漵牘蝟癆癇睞瞼禪篳箏縶緙緗緹緲繢緦緶緱縋緡臠醃蕆蕢蔞蛺蟯螄蠐褳襝襉覿觴謨讜謖"
  "諡謐齎賡賧躒輦輞輟輜鐒錸鋱鏗鋰鋥鋯鋨銼鋝鋶鐦鐗鋃鋟鋦錒闃闋靂靚頡頜潁頦颶饗餷餿騭騖魷魴鵓鸝鵠鵒鷳鵜黿縉囁噯嬡嬪尷攄欖櫬櫚櫸轂氳潷灩灄瀅癉磧磣稟穌竇縟縝縞縭縊縑繽"
  "耮齶靦膃蘺蕷鎣驀覦謫譾蹺蹕躚躋輳轡錆鍩錛鍀錁錕錮錈鍃錇錟錙闔闐闕韙韞頷颼饃饈驁騮騫騸鮁鮃鮎鱸鮒鱟鮐鵡鶓鵪鵯鶉齟齙齠廝嚶嬙戩攖曖檳櫧殯瀠瀟瀲瀦璦瘞瘻窶簀篋籜簞簫糉"
  "糝縹縵縲繆繅罌羆臏蘞藺蟈褸覯譖譙譎賻釅釃鑾鍥鍇鏘鍶鍔鍤鍾鎪鍰鎄鏤鐨鎇闞霽韜饉驃驄鮭鮚鮪鮞鱭鮫鯗鱘鶘鶚鶻鶿鶥鶩麼齜齦嚕屨襆擷擼攛檣櫫瓔簣餱餈纈繚繒耬聵蘄螻蠑襤覲覷"
  "觶讞譫賾躓躑轆鏌鎘鎸鎿鎦鎰鎵鑌靨韃鞽顎顓饜饊饌驏魘鯁鱺鰱鰹鰣鰷鯀鯊鯇鯽鷂鶼齏齬齪廩懍斕櫓櫞氌瀨癮癭穡繮繾繰繯藪蟎贗轔鏨鏢鏜鏝鏍鏞鏑鏃鏇顳顢顙飆飈魎鯖鯪鯫鯡鯤鯧鯝"
  "鯢鮎鯛鯴鯔鸚鷓鷚鹺黌嬤懣檁籪羈羶蘚躡蹣鐔钁鏷鑥鐓鑭鐠鑹鏹鐙鱝鰈鱷鰍鰒鰉鯿鰠鷯鷦鷲鷸齷囅癩鑊鐲鐿讎韉顥髏鰲鰭鰨鰥鰩鷺鸌巔籟纘讖躕鑔靄韝驥髖髕鰳鰾鱈鰻鰵鱅鞀瓚櫱鑣顬"
  "驤鬢鱖鱔鱒黷黲鼉灝癲躪顰鱧鸛趲躦鼴齇饢戇裡迴贊塼厰廼韆乾為昇厛厤伕紥鼕衹檯処嬭佈倣眾偽関沍氹弔喫郃嚮囘姦儘嵗戯儅釦粬硃懽襍縴攷堦貟祐躰彿剋鉋毉獃聼睏阬阯墰垻媯侷"
  "匲娬徬誌摺桿溈氾霛係蕓囌証穀閑靭敺暘珮璵姪凴颳厠捲週哢囯喦崠倖絃唸徴牴枴盃崐搆闆訢洩遝註霑氂麅綫儸捨膁槼錶眎迆郤埰隷牐閙擧玆脩勛噅敍偺嚥鬨嘩坰薑卹縂懕玨瑉竪毧鞦"
  "揹衚盪廕葯螘鐘籥鈎靣滻濜餄餎頫闓塚塤孃唕傢蓆噁輓梱傚覈棬慇凃濬畱貍癥綉臟艷蓧涖緻蒓賍賮缽鉆鉄騐偸鴞勗啗嚙菴綵慙慼慾凟釬畧毬瑯槃覜墝牋纍勣緔綳蔆啣訦醖剷舘蔴礄諝璡"
  "囉餜僳賸隄餵崳摣搇蒐稜箠匵櫂燄佘緍潟襇賫蹠煇舖耡銹鉲鐧濶騗餽鵐勦翬曡尲慴湣瘉擕煖霤灧谿灕煆炤硶痺篠簽懞簑辤酧鋻醯鍁穨躂墻鄘搾旂縯糢橰琍瘺堿耑琯燻镟篛甖衊踡痠鍫鋂"
  "譆璿澂覰錼飃骾鮌鎔譟嬾讌讚贋彫鯰贇簷燐繈鐝鰌鰐嚻籐繙鋜騣儹燿髩鱓臒";

const simpToTradMap = new Map<string, string>();
const tradToSimpMap = new Map<string, string>();
(() => {
  for (let i = 0; i < dictSimple.length; i++) {
    simpToTradMap.set(dictSimple[i], dictTrad[i]);
    tradToSimpMap.set(dictTrad[i], dictSimple[i]);
  }
})();

export function tradToSimple(text: string): string {
  if (!text) return '';
  let res = '';
  for (const ch of text) {
    res += tradToSimpMap.get(ch) ?? ch;
  }
  return res;
}

const fullSpaceCode = 12288;
const fullAsciiStart = 65281;
const fullAsciiEnd = 65374;

const fullPunctuations = new Map<string, string>([
  ['。', '.'],
  ['～', '~'],
  ['─', '-'],
  ['・', '·'],
  ['【', '['],
  ['】', ']'],
  ['“', '"'],
  ['”', '"'],
  ['‘', "'"],
  ['’', "'"],
  ['、', ',']
]);

export function fullToHalf(text: string, { punctuation = false } = {}): string {
  if (!text) return '';
  let res = '';
  for (const char of text) {
    const punct = punctuation ? fullPunctuations.get(char) : undefined;
    if (punct) {
      res += punct;
      continue;
    }
    const code = char.charCodeAt(0);
    if (code === fullSpaceCode) {
      res += char;
    } else if (fullAsciiStart <= code && code <= fullAsciiEnd) {
      res += String.fromCharCode(code - 65248);
    } else {
      res += char;
    }
  }
  return res;
}

export function normalizeTitle(title: string): string {
  return fullToHalf(tradToSimple(title), { punctuation: true });
}

// --- parseURLSearch / stringifyURLSearch ---

function coerceNumber(v: string | null): number | undefined {
  if (v === null || v === '') return undefined;
  const n = Number(v);
  return Number.isNaN(n) ? undefined : n;
}

function coerceDate(v: string | null): Date | undefined {
  if (v === null || v === '') return undefined;
  const n = Number(v);
  if (!Number.isNaN(n)) return new Date(n);
  const d = new Date(v);
  return Number.isNaN(d.getTime()) ? undefined : d;
}

export function parseURLSearch(
  params: URLSearchParams,
  body?: Partial<FilterOptions & PaginationOptions>
): { pagination: ResolvedPaginationOptions; filter: ResolvedFilterOptions } {
  const get = (k: string) => params.get(k);
  const getAll = (k: string) => params.getAll(k);

  const pagination: ResolvedPaginationOptions = {
    page: coerceNumber(get('page')) ?? body?.page ?? 1,
    pageSize: coerceNumber(get('pageSize')) ?? body?.pageSize ?? DefaultPageSize
  };
  if (Number.isNaN(pagination.page) || pagination.page < 1) pagination.page = 1;
  else pagination.page = Math.round(pagination.page);
  if (
    Number.isNaN(pagination.pageSize) ||
    pagination.pageSize < 1 ||
    pagination.pageSize > MaxRequestPageSize
  ) {
    pagination.pageSize = DefaultPageSize;
  } else {
    pagination.pageSize = Math.round(pagination.pageSize);
  }

  const filter: ResolvedFilterOptions = {};
  const providerParam = get('provider');
  const providerBody = body?.provider;
  if (providerBody) {
    filter.provider = providerBody as any;
    filter.duplicate = body?.duplicate ?? (get('duplicate') !== null ? coerceBool(get('duplicate')!) : true);
  } else if (providerParam) {
    filter.provider = providerParam as any;
    filter.duplicate = body?.duplicate ?? (get('duplicate') !== null ? coerceBool(get('duplicate')!) : true);
  }

  if (body?.fansub) filter.fansubs = [body.fansub];
  else if (body?.fansubs?.length) filter.fansubs = body.fansubs;
  else if (getAll('fansub').length) filter.fansubs = getAll('fansub');
  if (filter.fansubs) filter.fansubs = [...new Set(filter.fansubs)];

  if (body?.publisher) filter.publishers = [body.publisher];
  else if (body?.publishers?.length) filter.publishers = body.publishers;
  else if (getAll('publisher').length) filter.publishers = getAll('publisher');
  if (filter.publishers) filter.publishers = [...new Set(filter.publishers)];

  if (body?.type) filter.types = [body.type];
  else if (body?.types?.length) filter.types = body.types;
  else if (getAll('type').length) filter.types = getAll('type');
  if (filter.types) filter.types = [...new Set(filter.types)];

  const before = coerceDate(get('before')) ?? body?.before;
  if (before) filter.before = before;
  const after = coerceDate(get('after')) ?? body?.after;
  if (after) filter.after = after;

  if (body?.subject !== undefined) filter.subjects = [body.subject];
  else if (body?.subjects?.length) filter.subjects = body.subjects;
  else if (getAll('subject').length) {
    filter.subjects = getAll('subject').map((v) => Number(v)).filter((n) => !Number.isNaN(n));
  }
  if (filter.subjects) filter.subjects = [...new Set(filter.subjects)];

  if (body?.search?.length) filter.search = Array.isArray(body.search) ? body.search : [body.search];
  else if (getAll('search').length) filter.search = getAll('search');
  if (filter.search) filter.search = [...new Set(filter.search)];

  if (body?.include?.length) filter.include = Array.isArray(body.include) ? body.include : [body.include];
  else if (getAll('include').length) filter.include = getAll('include');
  if (filter.include) filter.include = [...new Set(filter.include)];

  if (body?.keywords?.length) filter.keywords = Array.isArray(body.keywords) ? body.keywords : [body.keywords];
  else if (getAll('keyword').length) filter.keywords = getAll('keyword');
  if (filter.keywords) filter.keywords = [...new Set(filter.keywords)];

  if (body?.exclude?.length) filter.exclude = Array.isArray(body.exclude) ? body.exclude : [body.exclude];
  else if (getAll('exclude').length) filter.exclude = getAll('exclude');
  if (filter.exclude) filter.exclude = [...new Set(filter.exclude)];

  if (body?.preset) filter.preset = body.preset as any;
  else if (get('preset')) filter.preset = get('preset') as any;

  if (filter.search) {
    delete filter.include;
  }

  return { pagination, filter };
}

function coerceBool(v: string): boolean {
  return Boolean(v);
}

export function stringifyURLSearch(options: any): URLSearchParams {
  const params = new URLSearchParams();

  const { page, pageSize, duplicate, after, before, preset } = options;
  if (preset) params.set('preset', preset);
  if (page) params.set('page', '' + page);
  if (pageSize) params.set('pageSize', '' + pageSize);
  if (duplicate) params.set('duplicate', 'true');
  if (after) {
    const time = after instanceof Date ? after.getTime() : new Date(after).getTime();
    params.set('after', '' + time);
  }
  if (before) {
    const time = before instanceof Date ? before.getTime() : new Date(before).getTime();
    params.set('before', '' + time);
  }
  if (options.provider) params.set('provider', options.provider);

  const { search, include, keywords, exclude, subject, subjects } = options;

  if (subject) {
    params.set('subject', '' + subject);
  } else if (subjects) {
    for (const s of new Set(subjects)) params.append('subject', '' + s);
  }

  if (search && search.length > 0) {
    for (const w of new Set(search)) params.append('search', String(w));
    for (const w of keywords ? new Set(keywords) : []) params.append('keyword', String(w));
    for (const w of exclude ? new Set(exclude) : []) params.append('exclude', String(w));
  } else if (include && include.length > 0) {
    for (const w of new Set(include)) params.append('include', String(w));
    for (const w of keywords ? new Set(keywords) : []) params.append('keyword', String(w));
    for (const w of exclude ? new Set(exclude) : []) params.append('exclude', String(w));
  } else {
    for (const w of keywords ? new Set(keywords) : []) params.append('keyword', String(w));
    for (const w of exclude ? new Set(exclude) : []) params.append('exclude', String(w));
  }

  const { type, types } = options;
  if (type) params.set('type', type);
  else if (types) {
    for (const t of new Set(types)) params.append('type', String(t));
  }

  const { fansub, fansubs } = options;
  if (fansub) params.set('fansub', fansub);
  else if (fansubs) {
    for (const f of new Set(fansubs)) params.append('fansub', String(f));
  }

  const { publisher, publishers } = options;
  if (publisher) params.set('publisher', publisher);
  else if (publishers) {
    for (const p of new Set(publishers)) params.append('publisher', String(p));
  }

  params.sort();
  return params;
}

// --- fetch helpers ---

export interface FetchResourcesResult {
  ok: boolean;
  resources: Resource[];
  pagination?: { page: number; pageSize: number; complete: boolean };
  filter?: ResolvedFilterOptions;
  timestamp?: Date;
  error?: any;
}

export async function fetchResources(
  options: (Partial<FilterOptions & PaginationOptions & ResolvedFilterOptions>) & {
    tracker?: boolean;
    metadata?: boolean;
  } = {}
): Promise<FetchResourcesResult> {
  const searchParams = stringifyURLSearch(options);
  if (options.tracker) searchParams.set('tracker', 'true');
  if (options.metadata) searchParams.set('metadata', 'true');

  try {
    const resp = await fetch(`/resources?${searchParams.toString()}`);
    const timestampHeader = resp.headers.get('X-Response-Timestamp');
    const data = await resp.json();
    if (data.status !== 'OK') {
      return { ok: false, resources: [], error: data };
    }
    return {
      ok: true,
      resources: (data.resources ?? []).map((r: any) => ({
        ...r,
        createdAt: new Date(r.createdAt),
        fetchedAt: new Date(r.fetchedAt)
      })),
      pagination: data.pagination,
      filter: data.filter,
      timestamp: timestampHeader ? new Date(timestampHeader) : undefined
    };
  } catch (error) {
    return { ok: false, resources: [], error };
  }
}

export async function fetchResourceDetail(
  provider: string,
  providerId: string
): Promise<{
  ok: boolean;
  resource?: Resource;
  detail?: ResourceDetail;
  isDeleted?: boolean;
  duplicatedId?: number;
  error?: any;
}> {
  try {
    const resp = await fetch(`/detail/${provider}/${encodeURIComponent(providerId)}`);
    const data = await resp.json();
    if (data.status !== 'OK') {
      return { ok: false, error: data };
    }
    return {
      ok: true,
      resource: data.resource
        ? { ...data.resource, createdAt: new Date(data.resource.createdAt), fetchedAt: new Date(data.resource.fetchedAt) }
        : undefined,
      detail: data.detail,
      isDeleted: data.isDeleted,
      duplicatedId: data.duplicatedId
    };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function fetchSubjects(): Promise<Subject[]> {
  try {
    const resp = await fetch('/subjects');
    const data = await resp.json();
    if (data.status === 'OK') {
      return (data.subjects ?? []).map((s: any) => ({ ...s, activedAt: new Date(s.activedAt) }));
    }
  } catch {}
  return [];
}

export interface CollectionItem {
  name: string;
  searchParams: string;
  [key: string]: any;
}

export interface Collection {
  name: string;
  authorization: string;
  filters: CollectionItem[];
}

export async function generateCollection(collection: Collection) {
  try {
    const resp = await fetch('/collection', {
      method: 'PUT',
      body: JSON.stringify({
        ...collection,
        filters: collection.filters.map((f) => ({
          ...f,
          resources: undefined,
          complete: undefined
        }))
      })
    });
    const data = await resp.json();
    if (data.status === 'OK') {
      return { ok: true, hash: data.hash, createdAt: data.createdAt, timestamp: new Date() };
    }
    return { ok: false, error: data };
  } catch (error) {
    return { ok: false, error };
  }
}

export async function fetchCollection(hash: string) {
  try {
    const resp = await fetch(`/collection/${hash}`);
    const data = await resp.json();
    if (data.status === 'OK') {
      return { ok: true, ...data };
    }
    return { ok: false, error: data };
  } catch (error) {
    return { ok: false, error };
  }
}

export function transformResourceHref(provider: string, href?: string) {
  switch (provider) {
    case 'dmhy':
      return `https://share.dmhy.org/topics/view/${href}`;
    case 'mikan':
      return `https://mikanani.me/Home/Episode/${href}`;
    case 'moe':
      return `https://bangumi.moe/torrent/${href}`;
    case 'ani':
      return href;
    default:
      return undefined;
  }
}

// Port of the original web parseSize: the API size field is in KB units.
export function parseSize(num: number) {
  if (num === 0) return '';
  if (num < 1024) return `${num} KB`;
  if (num < 1024 * 1024) return `${(num / 1024).toFixed(2)} MB`;
  return `${(num / 1024 / 1024).toFixed(2)} GB`;
}
