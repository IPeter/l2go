package main

import (
	"testing"
	"testing/fstest"
)

func TestReadClanHallData(t *testing.T) {
	fsys := fstest.MapFS{
		clanHallSQL: file("CREATE TABLE IF NOT EXISTS `clanhall` (`id` int(11) NOT NULL DEFAULT '0', `name` varchar(40) NOT NULL DEFAULT ''," +
			" `ownerId` int(11) NOT NULL DEFAULT '0', `lease` int(10) NOT NULL DEFAULT '0', `desc` text NOT NULL, `location` varchar(15) NOT NULL DEFAULT ''," +
			" `paidUntil` bigint(13) unsigned NOT NULL DEFAULT '0', `Grade` decimal(1,0) NOT NULL DEFAULT '0', `paid` int( 1 ) NOT NULL DEFAULT '0', PRIMARY KEY `id` (`id`));\n" +
			"INSERT IGNORE INTO `clanhall` VALUES ('31','The Atramental Barracks','0','200000','Clan hall located in the Town of Dion','Dion','0','1','0')," +
			"('22','Moonstone Hall','0','500000','Clan hall located in the Town of Gludio','Gludio','0','2','0');"),
		siegableClanHallSQL: file("CREATE TABLE IF NOT EXISTS `siegable_clanhall` (`clanHallId` int(10) NOT NULL DEFAULT '0', `name` varchar(45) DEFAULT NULL," +
			" `ownerId` int(10) DEFAULT NULL, `desc` varchar(100) DEFAULT NULL, `location` varchar(100) DEFAULT NULL, `nextSiege` bigint(20) DEFAULT NULL," +
			" `siegeLength` int(10) DEFAULT NULL, `schedule_config` varchar(20) DEFAULT NULL, PRIMARY KEY (`clanHallId`));\n" +
			"INSERT IGNORE INTO `siegable_clanhall` (`clanHallId`, `name`, `ownerId`, `desc`, `location`, `nextSiege`, `siegeLength`, `schedule_config`) VALUES" +
			" (21, 'Fortress of Resistance', 0, 'Contestable Clan Hall', 'Dion', 0, 3600000, '14;0;0;12;00');"),
		clanHallGuardsSQL: file("DROP TABLE IF EXISTS `clanhall_siege_guards`;\nCREATE TABLE IF NOT EXISTS `clanhall_siege_guards` (`clanHallId` tinyint(2) unsigned NOT NULL DEFAULT '0'," +
			" `npcId` smallint(5) unsigned NOT NULL DEFAULT '0', `x` mediumint(6) NOT NULL DEFAULT '0', `y` mediumint(6) NOT NULL DEFAULT '0', `z` mediumint(6) NOT NULL DEFAULT '0'," +
			" `heading` mediumint(6) NOT NULL DEFAULT '0', `respawnDelay` mediumint(5) NOT NULL DEFAULT '0', `isSiegeBoss` enum('false','true') NOT NULL DEFAULT 'false');\n" +
			"INSERT INTO `clanhall_siege_guards` VALUES (21,35375,45109,112124,-1900,0,10800,'true'), (21,35376,45000,112000,-1900,100,0,'false');"),
		auctionSQL: file("CREATE TABLE IF NOT EXISTS `auction` (`id` int(11) NOT NULL DEFAULT '0', `sellerId` int(11) NOT NULL DEFAULT '0'," +
			" `sellerName` varchar(50) NOT NULL DEFAULT 'NPC', `sellerClanName` varchar(50) NOT NULL DEFAULT '', `itemType` varchar(25) NOT NULL DEFAULT ''," +
			" `itemId` int(11) NOT NULL DEFAULT '0', `itemObjectId` int(11) NOT NULL DEFAULT '0', `itemName` varchar(40) NOT NULL DEFAULT ''," +
			" `itemQuantity` BIGINT UNSIGNED NOT NULL DEFAULT 0, `startingBid` BIGINT UNSIGNED NOT NULL DEFAULT 0, `currentBid` BIGINT UNSIGNED NOT NULL DEFAULT 0," +
			" `endDate` bigint(13) unsigned NOT NULL DEFAULT '0', PRIMARY KEY (`itemType`,`itemId`,`itemObjectId`), KEY `id` (`id`));"),
		clanHallAuctionsJava: file(`class ClanHallAuctionManager {
	private static final String[] ITEM_INIT_DATA =
	{
		"(31, 0, 'NPC', 'NPC Clan', 'ClanHall', 31, 0, 'The Atramental Barracks', 1, 8000000, 0, 1073037600000)", // Dion
		"(22, 0, 'NPC', 'NPC Clan', 'ClanHall', 22, 0, 'Moonstone Hall', 1, 20000000, 0, 1073037600000)"
	};
}`),
	}
	d, err := readClanHallData(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Halls) != 3 || d.Halls[0].ID != 21 || d.Halls[1].ID != 22 || d.Halls[2].ID != 31 {
		t.Fatalf("halls %+v, want 21, 22, 31", d.Halls)
	}
	if h := d.Halls[0]; !h.Siegable || h.SiegeLength != 3600000 || h.Schedule != [5]int{14, 0, 0, 12, 0} || h.Location != "Dion" {
		t.Errorf("Fortress of Resistance %+v", h)
	}
	if h := d.Halls[2]; h.Siegable || h.Lease != 200000 || h.Grade != 1 || h.Desc != "Clan hall located in the Town of Dion" {
		t.Errorf("Atramental Barracks %+v", h)
	}
	if len(d.Guards) != 2 || !d.Guards[0].SiegeBoss || d.Guards[0].RespawnDelay != 10800 || d.Guards[1].SiegeBoss {
		t.Errorf("guards %+v", d.Guards)
	}
	if len(d.Auctions) != 2 || d.Auctions[0].HallID != 22 || d.Auctions[0].StartingBid != 20000000 || d.Auctions[1].ItemName != "The Atramental Barracks" ||
		d.Auctions[1].EndDate != 1073037600000 {
		t.Errorf("auctions %+v", d.Auctions)
	}
}

func TestParseHallSchedule(t *testing.T) {
	if got, err := parseHallSchedule("7;1;0;20;30"); err != nil || got != [5]int{7, 1, 0, 20, 30} {
		t.Errorf("parseHallSchedule = %v, %v", got, err)
	}
	if _, err := parseHallSchedule("7;0;12"); err == nil {
		t.Error("a short schedule parsed")
	}
}
