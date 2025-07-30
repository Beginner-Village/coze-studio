package vo

type Env string

const (
	Draft  Env = "draft"
	Online Env = "online"
)

type CreateConversationTemplateMeta struct {
	UserID  int64
	AppID   int64
	SpaceID int64
	Name    string
}

type GetConversationTemplatePolicy struct {
	AppID      *int64
	Name       *string
	Version    *string
	TemplateID *int64
}

type ListConversationTemplatePolicy struct {
	AppID    int64
	Page     *Page
	NameLike *string
	Version  *string
}

type ListConversationMeta struct {
	APPID       int64
	UserID      int64
	ConnectorID int64
}

type ListConversationPolicy struct {
	ListConversationMeta

	Page     *Page
	NameLike *string
	Version  *string
}

type CreateStaticConversation struct {
	AppID       int64
	UserID      int64
	ConnectorID int64

	TemplateID int64
}
type CreateDynamicConversation struct {
	AppID       int64
	UserID      int64
	ConnectorID int64

	Name string
}
