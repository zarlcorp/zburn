package identity

// defaultDomain is the fallback domain when none is provided.
const defaultDomain = "zburn.id"

var firstNames = []string{
	"James", "Mary", "Robert", "Patricia", "John", "Jennifer", "Michael", "Linda",
	"David", "Elizabeth", "William", "Barbara", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Charles", "Karen", "Christopher", "Lisa", "Daniel", "Nancy",
	"Matthew", "Betty", "Anthony", "Margaret", "Mark", "Sandra", "Donald", "Ashley",
	"Steven", "Kimberly", "Paul", "Emily", "Andrew", "Donna", "Joshua", "Michelle",
	"Kenneth", "Carol", "Kevin", "Amanda", "Brian", "Dorothy", "George", "Melissa",
	"Timothy", "Deborah", "Ronald", "Stephanie", "Edward", "Rebecca", "Jason", "Sharon",
	"Jeffrey", "Laura", "Ryan", "Cynthia", "Jacob", "Kathleen", "Gary", "Amy",
	"Nicholas", "Angela", "Eric", "Shirley", "Jonathan", "Anna", "Stephen", "Brenda",
	"Larry", "Pamela", "Justin", "Emma", "Scott", "Nicole", "Brandon", "Helen",
	"Benjamin", "Samantha", "Samuel", "Katherine", "Raymond", "Christine", "Gregory", "Debra",
	"Frank", "Rachel", "Alexander", "Carolyn", "Patrick", "Janet", "Jack", "Catherine",
	"Dennis", "Maria", "Jerry", "Heather",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas",
	"Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson", "White",
	"Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson", "Walker", "Young",
	"Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
	"Green", "Adams", "Nelson", "Baker", "Hall", "Rivera", "Campbell", "Mitchell",
	"Carter", "Roberts", "Gomez", "Phillips", "Evans", "Turner", "Diaz", "Parker",
	"Cruz", "Edwards", "Collins", "Reyes", "Stewart", "Morris", "Morales", "Murphy",
	"Cook", "Rogers", "Gutierrez", "Ortiz", "Morgan", "Cooper", "Peterson", "Bailey",
	"Reed", "Kelly", "Howard", "Ramos", "Kim", "Cox", "Ward", "Richardson",
	"Watson", "Brooks", "Chavez", "Wood", "James", "Bennett", "Gray", "Mendoza",
	"Ruiz", "Hughes", "Price", "Alvarez", "Castillo", "Sanders", "Patel", "Myers",
	"Long", "Ross", "Foster", "Jimenez",
}

var cities = []string{
	"New York", "Los Angeles", "Chicago", "Houston", "Phoenix",
	"Philadelphia", "San Antonio", "San Diego", "Dallas", "San Jose",
	"Austin", "Jacksonville", "Fort Worth", "Columbus", "Indianapolis",
	"Charlotte", "San Francisco", "Seattle", "Denver", "Nashville",
	"Oklahoma City", "El Paso", "Boston", "Portland", "Las Vegas",
	"Memphis", "Louisville", "Baltimore", "Milwaukee", "Albuquerque",
	"Tucson", "Fresno", "Sacramento", "Mesa", "Kansas City",
	"Atlanta", "Omaha", "Raleigh", "Miami", "Minneapolis",
	"Tampa", "New Orleans", "Cleveland", "Pittsburgh", "Cincinnati",
	"St. Louis", "Orlando", "Richmond", "Salt Lake City", "Honolulu",
}

var states = []string{
	"AL", "AK", "AZ", "AR", "CA", "CO", "CT", "DE", "FL", "GA",
	"HI", "ID", "IL", "IN", "IA", "KS", "KY", "LA", "ME", "MD",
	"MA", "MI", "MN", "MS", "MO", "MT", "NE", "NV", "NH", "NJ",
	"NM", "NY", "NC", "ND", "OH", "OK", "OR", "PA", "RI", "SC",
	"SD", "TN", "TX", "UT", "VT", "VA", "WA", "WV", "WI", "WY",
}

var streetNames = []string{
	"Main", "Oak", "Maple", "Cedar", "Elm", "Pine", "Walnut", "Lake",
	"Hill", "Washington", "Park", "River", "Spring", "Church", "High",
	"Meadow", "Forest", "Sunset", "Valley", "Highland", "Lincoln",
	"Willow", "Birch", "Jackson", "Madison", "Franklin", "Jefferson",
	"Adams", "Monroe", "Cherry", "Chestnut", "Dogwood", "Magnolia",
	"Poplar", "Sycamore", "Linden", "Ash", "Beech", "Laurel", "Holly",
	"Ivy", "Rose", "Vine", "Peach", "Olive", "Market", "Broad",
	"Center", "Union", "Liberty",
}

var streetSuffixes = []string{
	"St", "Ave", "Blvd", "Dr", "Ln", "Ct", "Pl", "Way", "Rd", "Cir",
}

// adjectives for email generation
var adjectives = []string{
	"swift", "bold", "calm", "dark", "keen", "wild", "warm", "cool",
	"fast", "slow", "deep", "tall", "wide", "thin", "flat", "long",
	"soft", "hard", "pure", "rare", "safe", "fair", "fine", "free",
	"glad", "kind", "vast", "wise", "true", "pale", "gold", "iron",
	"blue", "gray", "jade", "ruby", "sage", "teal", "aqua", "mint",
	"dusk", "dawn", "moon", "star", "fern", "reed", "snow", "rain",
	"haze", "glow",
}

// nouns for email generation
var nouns = []string{
	"wolf", "hawk", "bear", "deer", "lynx", "fox", "owl", "crow",
	"pike", "bass", "wren", "dove", "lark", "swan", "moth", "wasp",
	"frog", "toad", "crab", "clam", "orca", "seal", "hare", "mole",
	"vole", "newt", "ibis", "kite", "jay", "ant", "bee", "ram",
	"oak", "elm", "ash", "bay", "fir", "yew", "ivy", "reed",
	"moss", "sage", "lily", "rose", "iris", "vine", "fern", "palm",
	"cliff", "ridge",
}
