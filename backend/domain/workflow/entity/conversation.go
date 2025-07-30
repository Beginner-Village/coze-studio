package entity

type ConversationTemplate struct {
	SpaceID    int64
	AppID      int64
	Name       string
	TemplateID int64
}

type StaticConversation struct {
	UserID         int64
	ConnectorID    int64
	TemplateID     int64
	ConversationID int64
}

type DynamicConversation struct {
	ID             int64
	UserID         int64
	ConnectorID    int64
	ConversationID int64
	Name           string
}
